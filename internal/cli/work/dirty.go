package work

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/pipe"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

const untrackedDiscardMessage = "Discard removes tracked edits only. These changes are untracked, so nothing would be discarded. Use --dirty=carry, --dirty=stash, or --dirty=cancel."

const missingDirtyChoice = "Uncommitted changes need an explicit choice: --dirty=carry, --dirty=stash, --dirty=discard, or --dirty=cancel."

const unbornDirtyMessage = "Uncommitted changes cannot be stashed because this repository has no commits yet."

var dirtyChoices = []prompt.Choice{
	{Value: "carry", Description: "Carry the changes onto the destination and keep the staged and unstaged split."},
	{Value: "stash", Description: "Stash the changes without carrying them."},
	{Value: "discard", Description: "Discard tracked edits only. Untracked files stay, and a collision is refused."},
	{Value: "cancel", Description: "Stop without changing working files."},
}

func addDirtyFlags(c *cobra.Command) {
	c.Flags().String("dirty", "", "How to handle uncommitted changes: carry, stash, discard, or cancel")
	c.Flags().String("stash-name", "", "Stash message for --dirty=stash. Scripts use this value, or a generated WIP name when it is omitted. In a terminal it is the suggestion and can still be edited.")
}

type navRequest struct {
	cmd    *cobra.Command
	ctx    context.Context
	id     cmdid.ID
	verb   string
	dest   string
	target func() (string, error)
	body   func() error
}

func applyNavigation(r navRequest) error {
	choice, name, nameSet, err := readDirtyFlags(r.cmd)
	if err != nil {
		return err
	}
	snap, err := state.ReadSnapshot()
	if err != nil {
		return err
	}
	if snap.Bare {
		return workflowText("This is a bare repository; it has no working tree.")
	}
	if snap.Operation != "" {
		return operationError(snap.Operation)
	}
	if snap.Unborn && snap.Relevant() {
		return unbornDirty(r, choice, snap)
	}
	if !snap.Relevant() {
		return checkoutUnlessIgnored(r, snap)
	}
	choice, err = chooseDirty(r, choice)
	if err != nil {
		return err
	}
	if choice == "cancel" {
		return cancelled(r.verb)
	}
	if err := refuseUnsupported(); err != nil {
		return err
	}
	label := ""
	if choice == "stash" {
		label, err = resolveStashLabel(r, name, nameSet)
		if err != nil {
			return err
		}
	}
	target := ""
	if r.target != nil {
		target, err = r.target()
		if err != nil {
			return err
		}
	}
	switch choice {
	case "discard":
		return discardThen(target, r.body)
	case "stash":
		if err := refusePreserve(target, false); err != nil {
			return err
		}
		kept, err := pipe.StashPipeNavigation(r.id, true, label, r.body)
		if err != nil {
			return err
		}
		if kept.OID == "" {
			return fmt.Errorf("stash handoff did not report a reachable stash. Inspect it with `eg repo doctor`")
		}
		reportStash(kept)
		return nil
	case "carry":
		if err := refusePreserve(target, true); err != nil {
			return err
		}
		_, err := pipe.StashPipeNavigation(r.id, false, "", r.body)
		return err
	default:
		return fmt.Errorf("invalid choice %q (use carry, stash, discard, or cancel)", choice)
	}
}

func readDirtyFlags(cmd *cobra.Command) (choice, name string, nameSet bool, err error) {
	if cmd == nil || cmd.Flags().Lookup("dirty") == nil {
		return "", "", false, nil
	}
	choice, err = cmd.Flags().GetString("dirty")
	if err != nil {
		return "", "", false, err
	}
	name, err = cmd.Flags().GetString("stash-name")
	if err != nil {
		return "", "", false, err
	}
	choice = strings.TrimSpace(choice)
	if f := cmd.Flags().Lookup("stash-name"); f != nil && f.Changed {
		nameSet = true
	}
	if choice == "" {
		if nameSet {
			return "", "", false, cliruntime.NewUsageError(cmd, fmt.Errorf("--stash-name requires --dirty=stash"))
		}
		return "", "", false, nil
	}
	switch choice {
	case "carry", "stash", "discard", "cancel":
	default:
		return "", "", false, cliruntime.NewUsageError(cmd, fmt.Errorf("--dirty=%s is not carry, stash, discard, or cancel", choice))
	}
	if nameSet && choice != "stash" {
		return "", "", false, cliruntime.NewUsageError(cmd, fmt.Errorf("--stash-name requires --dirty=stash"))
	}
	if nameSet && (strings.TrimSpace(name) == "" || strings.ContainsAny(name, "\r\n\x00")) {
		return "", "", false, cliruntime.NewUsageError(cmd, fmt.Errorf("stash name must be one non-blank line"))
	}
	return choice, strings.TrimSpace(name), nameSet, nil
}

func chooseDirty(r navRequest, explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	p := prompt.FromContext(r.ctx)
	if prompt.NonInteractive(p) {
		return "", workflowText(missingDirtyChoice)
	}
	ans, err := p.Pick(dirtyPrompt(r.dest), dirtyChoices, "carry")
	if err != nil {
		if errors.Is(err, prompt.ErrUserCancelled) || errors.Is(err, io.EOF) {
			return "", cancelled(r.verb)
		}
		return "", err
	}
	switch ans {
	case "carry", "stash", "discard", "cancel":
		return ans, nil
	default:
		return "", fmt.Errorf("invalid choice %q (use carry, stash, discard, or cancel)", ans)
	}
}

func dirtyPrompt(dest string) string {
	if dest == "" {
		dest = "the destination"
	}
	return "Uncommitted changes before " + dest + ". Discard removes tracked edits only and keeps untracked files"
}

func unbornDirty(r navRequest, choice string, snap state.Snapshot) error {
	if choice == "" && prompt.NonInteractive(prompt.FromContext(r.ctx)) {
		return workflowText(unbornDirtyMessage)
	}
	var err error
	if choice == "" {
		choice, err = chooseDirty(r, "")
		if err != nil {
			return err
		}
	}
	if choice == "cancel" {
		return cancelled(r.verb)
	}
	if choice == "discard" && !snap.HasTracked() {
		return workflowText(untrackedDiscardMessage)
	}
	return workflowText(unbornDirtyMessage)
}

func cancelled(verb string) error {
	return fmt.Errorf("%s cancelled", verb)
}

func refuseUnsupported() error {
	if err := pipe.UnsupportedWorktree(); err != nil {
		return workflowText(err.Error())
	}
	return nil
}

func resolveStashLabel(r navRequest, supplied string, suppliedSet bool) (string, error) {
	suggestion, err := suggestStashLabel()
	if err != nil {
		return "", err
	}
	p := prompt.FromContext(r.ctx)
	if prompt.NonInteractive(p) {
		if suppliedSet {
			return supplied, nil
		}
		return suggestion, nil
	}
	initial := suggestion
	if suppliedSet {
		initial = supplied
	}
	label, err := prompt.ValidatedLine(p, "Stash name", initial, func(s string) string {
		if strings.TrimSpace(s) == "" {
			return "stash name is required"
		}
		if strings.ContainsAny(s, "\r\n\x00") {
			return "stash name must be one line"
		}
		return ""
	})
	if err != nil {
		if errors.Is(err, prompt.ErrUserCancelled) || errors.Is(err, io.EOF) {
			return "", cancelled(r.verb)
		}
		return "", err
	}
	return strings.TrimSpace(label), nil
}

func suggestStashLabel() (string, error) {
	when := time.Now().Format("2006-01-02 15:04")
	ref, err := git.Output("symbolic-ref", "--quiet", "HEAD")
	ref = strings.TrimSpace(ref)
	if err == nil && ref != "" {
		return fmt.Sprintf("WIP on %s — %s", strings.TrimPrefix(ref, "refs/heads/"), when), nil
	}
	oid, err := git.Output("rev-parse", "--short", "HEAD")
	oid = strings.TrimSpace(oid)
	if err != nil || oid == "" {
		return "", fmt.Errorf("cannot name the stash")
	}
	return fmt.Sprintf("WIP at %s — %s", oid, when), nil
}

func reportStash(kept pipe.Retention) {
	where := kept.OID
	if kept.Selector != "" {
		where = kept.Selector + " " + kept.OID
	}
	text.InfoText(fmt.Sprintf("Stashed %q as %s.", kept.Label, where))
	text.InfoText("It was not applied. After you check out the branch you want, restore it with `git stash apply --index " + kept.OID + "`. Drop it only after that result is what you want.")
}

func discardThen(target string, body func() error) error {
	snap, err := state.ReadSnapshot()
	if err != nil {
		return err
	}
	if !snap.HasTracked() {
		return workflowText(untrackedDiscardMessage)
	}
	if path, err := blockedPath(target, preservedOccupants(snap, true), true); err != nil || path != "" {
		if err != nil {
			return err
		}
		return workflowText(fmt.Sprintf("%s would be overwritten. It was left in place.", path))
	}
	if err := git.Verbose("reset", "--hard", "HEAD"); err != nil {
		return err
	}
	again, err := state.ReadSnapshot()
	if err != nil {
		return err
	}
	if path, err := blockedPath(target, preservedOccupants(again, true), false); err != nil || path != "" {
		if err != nil {
			return err
		}
		return workflowText(fmt.Sprintf("%s would be overwritten. It was left in place. Tracked edits were already discarded, and checkout did not run.", path))
	}
	return body()
}

func checkoutUnlessIgnored(r navRequest, snap state.Snapshot) error {
	if len(snap.Ignored) == 0 {
		return r.body()
	}
	target := ""
	if r.target != nil {
		var err error
		target, err = r.target()
		if err != nil {
			return err
		}
	}
	path, err := blockedPath(target, snap.Ignored, false)
	if err != nil {
		return err
	}
	if path != "" {
		return workflowText(fmt.Sprintf("%s would be overwritten. It was left in place.", path))
	}
	return r.body()
}

func refusePreserve(target string, untracked bool) error {
	snap, err := state.ReadSnapshot()
	if err != nil {
		return err
	}
	if path, ok := nestedRepo(snap.Untracked); ok {
		return workflowText(fmt.Sprintf("A nested repository at %s cannot be carried or stashed. It was left in place.", path))
	}
	path, err := blockedPath(target, preservedOccupants(snap, untracked), false)
	if err != nil {
		return err
	}
	if path != "" {
		return workflowText(fmt.Sprintf("%s would be overwritten. It was left in place.", path))
	}
	return nil
}

func preservedOccupants(snap state.Snapshot, untracked bool) []string {
	if !untracked {
		return snap.Ignored
	}
	out := append([]string{}, snap.Untracked...)
	return append(out, snap.Ignored...)
}

func diffTarget(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(ref, "refs/") {
		return ref
	}
	if _, err := git.Output("rev-parse", "--verify", "--quiet", "refs/heads/"+ref+"^{commit}"); err == nil {
		return "refs/heads/" + ref
	}
	return ref
}

func blockedPath(target string, occupants []string, includeReset bool) (string, error) {
	var writes []string
	if includeReset {
		paths, err := diffPaths("HEAD")
		if err != nil {
			return "", err
		}
		writes = append(writes, paths...)
	}
	if target != "" {
		paths, err := diffPaths("HEAD", target)
		if err != nil {
			return "", err
		}
		writes = append(writes, paths...)
	}
	if path, ok := firstObstruction(writes, occupants, coreIgnoreCase()); ok {
		return path, nil
	}
	return "", nil
}

func diffPaths(revs ...string) ([]string, error) {
	args := []string{"diff", "--name-only", "--no-renames", "-z"}
	for _, rev := range revs {
		oid, err := git.Output("rev-parse", "--verify", "--quiet", rev+"^{commit}")
		if err != nil {
			msg := strings.TrimSpace(oid)
			if msg == "" {
				msg = err.Error()
			}
			return nil, fmt.Errorf("%s", msg)
		}
		args = append(args, strings.TrimSpace(oid))
	}
	args = append(args, "--")
	out, err := git.Output(args...)
	if err != nil {
		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s", msg)
	}
	if out == "" {
		return nil, nil
	}
	parts := strings.Split(out, "\x00")
	var paths []string
	for _, part := range parts {
		if part != "" {
			paths = append(paths, part)
		}
	}
	return paths, nil
}

func firstObstruction(writes, occupants []string, fold bool) (string, bool) {
	for _, occupant := range occupants {
		for _, write := range writes {
			if pathsCollide(write, occupant, fold) {
				return occupant, true
			}
		}
	}
	return "", false
}

func pathsCollide(write, occupant string, fold bool) bool {
	write = strings.TrimSuffix(write, "/")
	occupant = strings.TrimSuffix(occupant, "/")
	if write == "" || occupant == "" {
		return false
	}
	if fold {
		write = strings.ToLower(write)
		occupant = strings.ToLower(occupant)
	}
	if write == occupant {
		return true
	}
	return strings.HasPrefix(occupant, write+"/") || strings.HasPrefix(write, occupant+"/")
}

func nestedRepo(paths []string) (string, bool) {
	rootOut, err := git.Output("rev-parse", "--show-toplevel")
	root := strings.TrimSpace(rootOut)
	if err != nil || root == "" {
		return "", false
	}
	root = filepath.Clean(root)
	for _, path := range paths {
		path = strings.TrimSuffix(path, "/")
		if path == "" || filepath.IsAbs(path) || strings.Contains(path, "..") {
			continue
		}
		dir := path
		info, statErr := os.Stat(path)
		if statErr == nil && !info.IsDir() {
			dir = filepath.Dir(path)
		}
		for {
			full := filepath.Clean(filepath.Join(root, dir))
			if full == root || !strings.HasPrefix(full, root+string(filepath.Separator)) {
				break
			}
			if _, err := os.Stat(filepath.Join(full, ".git")); err == nil {
				return path, true
			}
			parent := filepath.Dir(dir)
			if parent == dir || parent == "." {
				break
			}
			dir = parent
		}
	}
	return "", false
}

func dirtyClass(object, action string) string {
	switch object + "." + action {
	case "work.start", "work.switch":
		return "prepare"
	case "work.save":
		return "consume"
	case "work.polish", "work.sync", "work.push", "work.accept", "release.new", "repo.prune":
		return "prepare-legacy"
	case "repo.init", "repo.clone":
		return "create"
	default:
		return "tolerate"
	}
}
