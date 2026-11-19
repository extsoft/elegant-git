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

const discardScope = "restores committed-file edits. New-path files stay unless you delete them"

const unknownIncoming = "The incoming changes cannot be identified, so nothing was stashed or reset."

const missingDirtyChoice = "Uncommitted changes need an explicit choice: --dirty=carry, --dirty=stash, --dirty=discard, or --dirty=cancel."

const unbornDirtyMessage = "Uncommitted changes cannot be stashed because this repository has no commits yet."

var dirtyChoices = []prompt.Choice{
	{Value: "carry", Description: "Carry the changes onto the destination and keep the staged and unstaged split."},
	{Value: "stash", Description: "Stash the changes without carrying them."},
	{Value: "discard", Description: "Restore committed-file edits. Keep new-path files, or delete the listed ones permanently with no backup."},
	{Value: "cancel", Description: "Stop without changing working files."},
}

// AddDirtyFlags registers the shared --dirty and --stash-name flags.
func AddDirtyFlags(c *cobra.Command) {
	addDirtyFlags(c)
}

func addDirtyFlags(c *cobra.Command) {
	c.Flags().String("dirty", "", "How to handle uncommitted changes: carry, stash, discard, or cancel")
	c.Flags().String("stash-name", "", "Stash message for --dirty=stash. Scripts use this value, or a generated WIP name when it is omitted. In a terminal it is the suggestion and can still be edited.")
	c.Flags().String("untracked", "", "With --dirty=discard, keep or delete new-path files, including staged additions and rename targets. Without a terminal, omit this to keep them. delete removes the listed files permanently and keeps no backup.")
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
	flags, err := readDirtyFlags(r.cmd)
	if err != nil {
		return err
	}
	choice, name, nameSet := flags.choice, flags.name, flags.nameSet
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
		return unbornDirty(r, choice)
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
	if err := guardChoice(r.ctx, choice); err != nil {
		return err
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
	if choice == "discard" {
		if err := bindDisposition(r.cmd, r.ctx, r.verb); err != nil {
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
		return discardThen(r.cmd, target, r.body)
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
		recordStash(r.ctx, kept)
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

func readDirtyFlags(cmd *cobra.Command) (dirtyFlags, error) {
	var flags dirtyFlags
	if cmd == nil || cmd.Flags().Lookup("dirty") == nil {
		return flags, nil
	}
	choice, err := cmd.Flags().GetString("dirty")
	if err != nil {
		return flags, err
	}
	name, err := cmd.Flags().GetString("stash-name")
	if err != nil {
		return flags, err
	}
	untracked := ""
	if cmd.Flags().Lookup("untracked") != nil {
		untracked, err = cmd.Flags().GetString("untracked")
		if err != nil {
			return flags, err
		}
		if f := cmd.Flags().Lookup("untracked"); f != nil && f.Changed {
			flags.untrackedSet = true
		}
	}
	choice = strings.TrimSpace(choice)
	untracked = strings.TrimSpace(untracked)
	if f := cmd.Flags().Lookup("stash-name"); f != nil && f.Changed {
		flags.nameSet = true
	}
	flags.choice = choice
	flags.name = strings.TrimSpace(name)
	flags.untracked = untracked
	if flags.untrackedSet && untracked != "keep" && untracked != "delete" {
		return flags, cliruntime.NewUsageError(cmd, fmt.Errorf("--untracked=%s is not keep or delete", untracked))
	}
	if flags.untrackedSet && choice != "discard" {
		return flags, cliruntime.NewUsageError(cmd, fmt.Errorf("--untracked requires --dirty=discard"))
	}
	if choice == "" {
		if flags.nameSet {
			return flags, cliruntime.NewUsageError(cmd, fmt.Errorf("--stash-name requires --dirty=stash"))
		}
		return flags, nil
	}
	switch choice {
	case "carry", "stash", "discard", "cancel":
	default:
		return flags, cliruntime.NewUsageError(cmd, fmt.Errorf("--dirty=%s is not carry, stash, discard, or cancel", choice))
	}
	if flags.nameSet && choice != "stash" {
		return flags, cliruntime.NewUsageError(cmd, fmt.Errorf("--stash-name requires --dirty=stash"))
	}
	if flags.nameSet && (strings.TrimSpace(name) == "" || strings.ContainsAny(name, "\r\n\x00")) {
		return flags, cliruntime.NewUsageError(cmd, fmt.Errorf("stash name must be one non-blank line"))
	}
	return flags, nil
}

// ValidateDirtyFlags reports a usage error for --dirty, --stash-name, or --untracked.
func ValidateDirtyFlags(cmd *cobra.Command) error {
	_, err := readDirtyFlags(cmd)
	return err
}

func chooseDirty(r navRequest, explicit string) (string, error) {
	if err := showDirtyPreview(r.dest); err != nil {
		return "", err
	}
	before, err := readIdentity()
	if err != nil {
		return "", err
	}
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
		after, err := readIdentity()
		if err != nil {
			return "", err
		}
		if !before.same(after) {
			return "", workflowText(changedBeforePrep)
		}
		return ans, nil
	default:
		return "", fmt.Errorf("invalid choice %q (use carry, stash, discard, or cancel)", ans)
	}
}

func dirtyPrompt(dest string) string {
	if dest == "" {
		dest = "the destination"
	}
	return "Uncommitted changes before " + dest + ". Discard " + discardScope
}

func unbornDirty(r navRequest, choice string) error {
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
	if err := guardChoice(r.ctx, choice); err != nil {
		return err
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

func revertTrackedEdits(keep, restore []string) error {
	if len(keep) == 0 && len(restore) == 0 {
		return nil
	}
	prefix, err := gitAtRoot()
	if err != nil {
		return err
	}
	if len(keep) > 0 {
		args := append(append([]string{}, prefix...), "--literal-pathspecs", "rm", "--cached", "-f", "--ignore-unmatch", "--")
		args = append(args, keep...)
		if err := git.Verbose(args...); err != nil {
			return err
		}
	}
	if len(restore) > 0 {
		args := append(append([]string{}, prefix...), "--literal-pathspecs", "restore", "--source=HEAD", "--staged", "--worktree", "--")
		args = append(args, restore...)
		if err := git.Verbose(args...); err != nil {
			return err
		}
	}
	return nil
}

func gitAtRoot() ([]string, error) {
	out, err := git.Output("rev-parse", "--show-prefix")
	if err != nil {
		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s", msg)
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}
	root, err := repoRoot()
	if err != nil {
		return nil, err
	}
	return []string{"-C", root}, nil
}

func discardPlan() (keep, restore []string, err error) {
	keep, restore, _, err = planTracked()
	if err != nil {
		return nil, nil, err
	}
	keep = uniquePaths(keep)
	kept := map[string]bool{}
	for _, path := range keep {
		kept[path] = true
	}
	var restoring []string
	seen := map[string]bool{}
	for _, path := range restore {
		if kept[path] || seen[path] {
			continue
		}
		seen[path] = true
		restoring = append(restoring, path)
	}
	return keep, restoring, nil
}

func uniquePaths(paths []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, path := range paths {
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	return out
}

func classifyNameStatus(out string) (keep, restore []string, kinds map[string]string, err error) {
	kinds = map[string]string{}
	parts := strings.Split(out, "\x00")
	for i := 0; i < len(parts); {
		status := parts[i]
		if status == "" {
			i++
			continue
		}
		switch status[:1] {
		case "R":
			if i+2 >= len(parts) {
				return nil, nil, nil, fmt.Errorf("cannot list tracked edits")
			}
			restore = append(restore, parts[i+1])
			keep = append(keep, parts[i+2])
			kinds[parts[i+2]] = "renamed"
			i += 3
		case "C":
			if i+2 >= len(parts) {
				return nil, nil, nil, fmt.Errorf("cannot list tracked edits")
			}
			keep = append(keep, parts[i+2])
			kinds[parts[i+2]] = "copied"
			i += 3
		case "A":
			if i+1 >= len(parts) {
				return nil, nil, nil, fmt.Errorf("cannot list tracked edits")
			}
			keep = append(keep, parts[i+1])
			kinds[parts[i+1]] = "added"
			i += 2
		case "M", "D", "T":
			if i+1 >= len(parts) {
				return nil, nil, nil, fmt.Errorf("cannot list tracked edits")
			}
			restore = append(restore, parts[i+1])
			i += 2
		default:
			return nil, nil, nil, fmt.Errorf("cannot list tracked edits")
		}
	}
	return keep, restore, kinds, nil
}

func incomingTarget(h History) (string, error) {
	if h.Resolve == nil {
		return "", workflowText(unknownIncoming)
	}
	target, err := h.Resolve()
	if err != nil {
		return "", err
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return "", workflowText(unknownIncoming)
	}
	if _, err := git.Output("rev-parse", "--verify", "--quiet", diffTarget(target)+"^{commit}"); err != nil {
		return "", workflowText(unknownIncoming)
	}
	return target, nil
}

func refuseIncoming(h History, choice, target string) error {
	snap, err := state.ReadSnapshot()
	if err != nil {
		return err
	}
	var writes []string
	if h.Footprint != nil {
		writes, err = h.Footprint(target)
	} else {
		writes, err = diffPaths("HEAD", target)
		if err == nil && h.Replay {
			var adds []string
			adds, err = replayAdds(target, "HEAD")
			writes = append(writes, adds...)
		}
	}
	if err != nil {
		return workflowText(unknownIncoming)
	}
	occupants := preservedOccupants(snap, choice == "carry")
	if choice == "discard" {
		occupants = preservedOccupants(snap, true)
		if DiscardRemoves(h.Cmd) {
			drop, err := DisposablePaths()
			if err != nil {
				return err
			}
			occupants = withoutPaths(occupants, drop)
		}
	}
	if h.Occupants != nil {
		occupants, err = h.Occupants(snap, choice)
		if err != nil {
			return err
		}
	}
	if path, ok := firstObstruction(writes, occupants, coreIgnoreCase()); ok {
		return workflowText(fmt.Sprintf("%s would be overwritten. It was left in place.", path))
	}
	return nil
}

// DiscardKeeps lists paths discard leaves in the worktree: staged additions,
// copy targets, and rename targets.
func DiscardKeeps() ([]string, error) {
	keep, _, err := discardPlan()
	return keep, err
}

// DiffPaths lists paths that differ between commits.
func DiffPaths(revs ...string) ([]string, error) {
	return diffPaths(revs...)
}

// ReplayAddedPaths lists files added between commits.
func ReplayAddedPaths(from, to string) ([]string, error) {
	return replayAdds(from, to)
}

// RefuseIgnored reports an ignored path the write set would overwrite.
func RefuseIgnored(paths []string) error {
	snap, err := state.ReadSnapshot()
	if err != nil {
		return err
	}
	if path, ok := firstObstruction(paths, snap.Ignored, coreIgnoreCase()); ok {
		return workflowText(fmt.Sprintf("%s would be overwritten. It was left in place.", path))
	}
	return nil
}

func replayAdds(from, to string) ([]string, error) {
	out, err := git.Output("log", "-z", "--format=", "--name-only", "--diff-filter=A", "--no-renames", "--no-merges", from+".."+to)
	if err != nil {
		return nil, err
	}
	return splitGitPaths(out), nil
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

var errHistoryDiscard = errors.New("discard tracked edits")

type recordedStash struct {
	kept pipe.Retention
}

type recordedStashKey struct{}

func withRecordedStash(ctx context.Context) (context.Context, *recordedStash) {
	rec := &recordedStash{}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, recordedStashKey{}, rec), rec
}

func recordStash(ctx context.Context, kept pipe.Retention) {
	if ctx == nil {
		return
	}
	rec, _ := ctx.Value(recordedStashKey{}).(*recordedStash)
	if rec == nil {
		return
	}
	rec.kept = kept
}

type choiceGuard func(string) error

type choiceGuardKey struct{}

func withChoiceGuard(ctx context.Context, guard choiceGuard) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, choiceGuardKey{}, guard)
}

func guardChoice(ctx context.Context, choice string) error {
	if ctx == nil {
		return nil
	}
	guard, _ := ctx.Value(choiceGuardKey{}).(choiceGuard)
	if guard == nil {
		return nil
	}
	return guard(choice)
}

// History runs one prepare command through the shared dirty choice.
type History struct {
	Cmd      *cobra.Command
	ID       cmdid.ID
	Verb     string
	Dest     string
	Allow    func(state.Snapshot) bool
	Mismatch func(state.Snapshot) error
	Protect  func() error
	// Resolve identifies the commit the body will rebase or merge onto. It may
	// fetch. A dirty tree is refused before preparation when this is missing
	// or the footprint cannot be computed.
	Resolve func() (string, error)
	// Replay includes paths added by commits the body replays onto Resolve.
	Replay bool
	// Footprint overrides the default HEAD-to-target write set.
	Footprint func(target string) ([]string, error)
	// Occupants overrides which paths block that write set. Nil keeps ignored
	// files, plus untracked files when the choice is carry.
	Occupants func(state.Snapshot, string) ([]string, error)
	// RestoreWhenSafe applies a carried backup after the body fails when HEAD
	// is on the recorded destination and no Git operation is in progress.
	RestoreWhenSafe bool
	Body            func() error
}

// PrepareHistory classifies an in-progress operation before protected checks
// and the dirty choice. Continuation and recovery stay on the shared pipe.
func PrepareHistory(h History) error {
	flags, err := readDirtyFlags(h.Cmd)
	explicit := flags.choice
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
		if h.Allow != nil && h.Allow(snap) {
			_, err = pipe.StashPipeHistory(h.ID, nil, h.Body, h.RestoreWhenSafe)
			return err
		}
		if h.Mismatch != nil {
			return h.Mismatch(snap)
		}
		return operationError(snap.Operation)
	}
	if len(snap.Unmerged) > 0 {
		return workflowText("backup preparation does not support unmerged index entries")
	}
	if snap.Unborn && snap.Relevant() {
		return workflowText(unbornDirtyMessage)
	}
	if h.Protect != nil {
		if err := h.Protect(); err != nil {
			return err
		}
	}
	ctx := context.Background()
	if h.Cmd != nil && h.Cmd.Context() != nil {
		ctx = h.Cmd.Context()
	}
	choice := ""
	incoming := ""
	kept, err := pipe.StashPipeHistory(h.ID, func() (bool, string, error) {
		picked, err := chooseDirty(navRequest{cmd: h.Cmd, ctx: ctx, verb: h.Verb, dest: h.Dest}, explicit)
		if err != nil {
			return false, "", err
		}
		choice = picked
		if picked == "cancel" {
			return false, "", cancelled(h.Verb)
		}
		var label string
		if picked == "stash" {
			name, nameSet, err := stashName(h.Cmd)
			if err != nil {
				return false, "", err
			}
			label, err = resolveStashLabel(navRequest{cmd: h.Cmd, ctx: ctx, verb: h.Verb, dest: h.Dest}, name, nameSet)
			if err != nil {
				return false, "", err
			}
		}
		target, err := incomingTarget(h)
		if err != nil {
			return false, "", err
		}
		incoming = target
		if picked == "discard" {
			if err := refuseUnsupported(); err != nil {
				return false, "", err
			}
			if err := bindDisposition(h.Cmd, ctx, h.Verb); err != nil {
				return false, "", err
			}
		}
		if err := refuseIncoming(h, picked, target); err != nil {
			return false, "", err
		}
		if picked == "discard" {
			return false, "", errHistoryDiscard
		}
		if err := refuseUnsupported(); err != nil {
			return false, "", err
		}
		if err := refusePreserve("", true); err != nil {
			return false, "", err
		}
		if picked == "stash" {
			return true, label, nil
		}
		return false, "", nil
	}, h.Body, h.RestoreWhenSafe)
	if errors.Is(err, errHistoryDiscard) {
		return discardThen(h.Cmd, incoming, h.Body)
	}
	if err != nil {
		return noteKeptBackup(err, kept)
	}
	if choice == "stash" {
		if kept.OID == "" {
			return fmt.Errorf("stash handoff did not report a reachable stash. Inspect it with `eg repo doctor`")
		}
		reportStash(kept)
	}
	return nil
}

func annotate(err error, msg string) error {
	if err == nil {
		return nil
	}
	var status cliruntime.ExitStatus
	if errors.As(err, &status) {
		text.ErrorText(msg + ".")
		return err
	}
	return fmt.Errorf("%s: %w", msg, err)
}

func noteKeptBackup(err error, kept pipe.Retention) error {
	if err == nil || kept.OID == "" {
		return err
	}
	var status cliruntime.ExitStatus
	if errors.As(err, &status) {
		text.ErrorText("The recovery backup was kept.")
		return err
	}
	return fmt.Errorf("%w\nThe recovery backup was kept.", err)
}

func stashName(cmd *cobra.Command) (string, bool, error) {
	flags, err := readDirtyFlags(cmd)
	return flags.name, flags.nameSet, err
}

func dirtyClass(object, action string) string {
	switch object + "." + action {
	case "work.start", "work.switch":
		return "prepare"
	case "work.save":
		return "consume"
	case "work.polish", "work.sync", "work.push", "work.accept", "release.new":
		return "prepare"
	case "repo.prune":
		return "prepare"
	case "repo.init", "repo.clone":
		return "create"
	default:
		return "tolerate"
	}
}
