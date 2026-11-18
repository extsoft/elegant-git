package repo

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/statefmt"
	"github.com/extsoft/elegant-git/internal/cli/work"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/exitcode"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/pipe"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/recovery"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var pruneID = cmdid.ID{Command: "repo", Action: "prune"}

const (
	pruneRefFormat   = "%(refname:short)%09%(committerdate:unix)%09%(committerdate:short)%09%(upstream:short)"
	pruneStashFormat = "%H%x09%gd%x09%ct%x09%cs%x09%s"
	pruneNameMaxLen  = 40
)

var pruneNowUnix = func() int64 { return time.Now().Unix() }

var prunePhaseNames = []string{"local", "remote", "stashes", "worktrees"}

func newPruneCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "prune [phase...]",
		Short: "Cleans up branches, stashes, and worktrees",
		Long: `Deletes local branches merged into the default development branch or a protected branch, or whose upstream is gone; deletes branches on origin merged into those same branches; lists ordinary stashes without dropping them; and prunes worktrees whose directory is gone. Pass phase names to choose. Interactive mode asks which phases to run before checkout, lets you pick candidates, and confirms before each phase applies. Non-interactive mode with no phases runs local only and applies without asking. The stashes phase lists ordinary stashes and prints how to drop one with git stash drop. It does not ask, and it works without a terminal.

When there are uncommitted changes, choose carry, stash, discard, or cancel before that checkout. carry restores them on the default development branch and keeps the staged and unstaged split. If prune deletes the branch you were on, the backup stays and is not applied there, and a later prune does not run again until eg repo doctor clears it. A phase that fails after checkout still restores a carried backup when HEAD is on the default branch and no rebase is in progress. stash leaves them in the stash instead, including when a later phase fails. The stash name is asked before any fetch. discard reverts modifications and deletions of committed files and leaves staged additions and rename targets as untracked files. An ignored file the checkout or rebase would write is refused before any stash or reset, and so is an untracked file for carry or discard, including a staged addition or rename target that carry or discard would leave behind. cancel stops before any of that. A non-interactive run with uncommitted changes fails unless --dirty is set. A clean tree does not need --dirty. An in-progress Git operation is refused before that choice. Cancelling a phase after preparation skips that phase and does not undo the checkout. Prune stays on the default branch.`,
		ValidArgs: append([]string(nil), prunePhaseNames...),
		Args:      cobra.OnlyValidArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, pruneID, func() error {
				return pruneRun(cmd, args)
			})
		},
	}
	work.AddDirtyFlags(c)
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func pruneRun(cmd *cobra.Command, args []string) error {
	if err := work.ValidateDirtyFlags(cmd); err != nil {
		return err
	}
	p := prompt.FromContext(cmd.Context())
	phases, err := prunePhases(p, args)
	if err != nil {
		return err
	}
	if len(phases) == 0 {
		return nil
	}
	base := config.DefaultBranch()
	plan := &prunePlan{base: base}
	return work.PrepareHistory(work.History{
		Cmd:             cmd,
		ID:              pruneID,
		Verb:            "repo prune",
		Dest:            "the default branch " + base,
		Occupants:       pruneOccupants,
		Resolve:         plan.resolve,
		Footprint:       func(string) ([]string, error) { return plan.writes() },
		RestoreWhenSafe: true,
		Body: func() error {
			if err := plan.prepare(); err != nil {
				return noteRemember(plan, err)
			}
			for _, phase := range phases {
				if err := prunePhase(cmd.OutOrStdout(), p, phase, base); err != nil {
					return noteRemember(plan, err)
				}
			}
			return plan.remember()
		},
	})
}

func noteRemember(plan *prunePlan, err error) error {
	if rerr := plan.remember(); rerr != nil {
		return errors.Join(err, rerr)
	}
	return err
}

func prunePhases(p prompt.Prompter, args []string) ([]string, error) {
	phases := orderedPhases(args)
	if len(args) != 0 {
		return phases, nil
	}
	if prompt.NonInteractive(p) {
		return []string{"local"}, nil
	}
	picked, err := prompt.PickMany(p, "What to clean up", phaseChoices(), append([]string(nil), prunePhaseNames...))
	if err != nil {
		return nil, err
	}
	return orderedPhases(picked), nil
}

type prunePlan struct {
	base    string
	branch  string
	fetched bool
	rebase  bool
}

func (p *prunePlan) resolve() (string, error) {
	if _, err := p.fetch(); err != nil {
		return "", err
	}
	if _, err := git.Output("rev-parse", "--verify", "--quiet", "refs/heads/"+p.base+"^{commit}"); err != nil {
		return "", err
	}
	return "refs/heads/" + p.base, nil
}

func (p *prunePlan) fetch() (bool, error) {
	if p.fetched {
		return p.rebase, nil
	}
	p.fetched = true
	if !state.IsThereUpstreamFor(p.base) {
		return false, nil
	}
	if err := git.Verbose("fetch", "--all", "--prune"); err != nil {
		text.InfoText("As the remotes can't be fetched, the current local version is used.")
		return false, nil
	}
	p.rebase = true
	return true, nil
}

func (p *prunePlan) writes() ([]string, error) {
	if _, err := p.fetch(); err != nil {
		return nil, err
	}
	writes, err := work.DiffPaths("HEAD", "refs/heads/"+p.base)
	if err != nil {
		return nil, err
	}
	if !p.rebase {
		return writes, nil
	}
	up := state.UpstreamOf(p.base)
	if up == "" {
		return nil, fmt.Errorf("upstream is not available")
	}
	more, err := work.DiffPaths("refs/heads/"+p.base, up)
	if err != nil {
		return nil, err
	}
	adds, err := work.ReplayAddedPaths(up, "refs/heads/"+p.base)
	if err != nil {
		return nil, err
	}
	return append(append(writes, more...), adds...), nil
}

func (p *prunePlan) guardClean() error {
	snap, err := state.ReadSnapshot()
	if err != nil {
		return err
	}
	if len(snap.Ignored) == 0 {
		return nil
	}
	paths, err := p.writes()
	if err != nil {
		return pruneWorkflow("The incoming changes cannot be identified, so nothing was stashed or reset.")
	}
	return work.RefuseIgnored(paths)
}

func (p *prunePlan) prepare() error {
	if err := p.guardClean(); err != nil {
		return err
	}
	if err := p.capture(); err != nil {
		return err
	}
	if err := git.Verbose("checkout", p.base); err != nil {
		return err
	}
	rebase, err := p.fetch()
	if err != nil {
		return err
	}
	if !rebase {
		return nil
	}
	return git.Verbose("rebase")
}

func (p *prunePlan) capture() error {
	ref, err := git.Output("symbolic-ref", "--quiet", "HEAD")
	ref = strings.TrimSpace(ref)
	if err == nil && ref != "" {
		p.branch = strings.TrimPrefix(ref, "refs/heads/")
		return nil
	}
	if _, err := git.Output("rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
		return err
	}
	return nil
}

func (p *prunePlan) remember() error {
	if p.branch != "" && !state.LocalBranchExists(p.branch) {
		return nil
	}
	return pipe.RememberRestoreBranch(p.base)
}

func pruneOccupants(snap state.Snapshot, choice string) ([]string, error) {
	out := append([]string{}, snap.Ignored...)
	if choice != "carry" && choice != "discard" {
		return out, nil
	}
	out = append(out, snap.Untracked...)
	keep, err := work.DiscardKeeps()
	if err != nil {
		return nil, err
	}
	return append(out, keep...), nil
}

func pruneWorkflow(msg string) error {
	text.ErrorText(msg)
	return cliruntime.ExitStatus{Code: exitcode.WorkflowError}
}

func phaseChoices() []prompt.Choice {
	return []prompt.Choice{
		{Value: "local", Description: "Delete local branches that are merged or whose upstream is gone."},
		{Value: "remote", Description: "Delete remote branches that are merged."},
		{Value: "stashes", Description: "List ordinary stashes. Elegant Git cannot drop one safely by commit."},
		{Value: "worktrees", Description: "Prune worktrees whose directory is gone."},
	}
}

func orderedPhases(names []string) []string {
	want := map[string]bool{}
	for _, name := range names {
		want[name] = true
	}
	var out []string
	for _, name := range prunePhaseNames {
		if want[name] {
			out = append(out, name)
		}
	}
	return out
}

const stashDropGuidance = "Elegant Git cannot drop one stash safely by commit. Drop one yourself with `git stash drop stash@{n}`."

func prunePhase(w io.Writer, p prompt.Prompter, phase, defaultBranch string) error {
	if phase == "stashes" {
		return listStashes(w)
	}
	text.CommandText(phaseHeader(phase, defaultBranch))
	progress := newPruneProgress(w, phaseNoun(phase))
	choices, stashOIDs, err := collectPhase(phase, defaultBranch, progress)
	progress.close()
	if err != nil {
		return err
	}
	if len(choices) == 0 {
		text.InfoText(emptyPhaseMessage(phase, defaultBranch))
		return nil
	}
	selected := selectedDefaults(phase, choices)
	if !prompt.NonInteractive(p) {
		selected, err = prompt.PickMany(p, phasePrompt(phase), choices, selected)
		if errors.Is(err, prompt.ErrUserCancelled) {
			text.InfoText(phaseNothingMessage(phase))
			return nil
		}
		if err != nil {
			return err
		}
		if len(selected) == 0 {
			text.InfoText(phaseNothingMessage(phase))
			return nil
		}
		ok, err := p.Confirm(phaseVerb(phase)+" "+phaseCount(phase, len(selected))+"?", false)
		if err != nil {
			return err
		}
		if !ok {
			text.InfoText(phaseNothingMessage(phase))
			return nil
		}
	}
	if len(selected) == 0 {
		return nil
	}
	if err := deletePhase(phase, choices, selected, stashOIDs); err != nil {
		return err
	}
	text.InfoText(phasePastVerb(phase) + " " + phaseCount(phase, len(selected)) + ".")
	return nil
}

func phaseHeader(phase, defaultBranch string) string {
	switch phase {
	case "local":
		return "Pruning local branches merged into " + defaultBranch + " or a protected branch, or whose upstream is gone..."
	case "remote":
		return "Pruning " + config.DefaultUpstreamRemote + " branches merged into " + defaultBranch + " or a protected branch (deletes them on " + config.DefaultUpstreamRemote + ")..."
	case "stashes":
		return "Listing ordinary stashes..."
	case "worktrees":
		return "Pruning worktrees whose directory no longer exists..."
	default:
		return "Pruning " + phase + "..."
	}
}

func phaseNoun(phase string) string {
	switch phase {
	case "local":
		return "local branches"
	case "remote":
		return "remote branches"
	case "stashes":
		return "stashes"
	case "worktrees":
		return "worktrees"
	default:
		return phase
	}
}

func phasePrompt(phase string) string {
	switch phase {
	case "local":
		return "Local branches to delete"
	case "remote":
		return "Branches to delete on " + config.DefaultUpstreamRemote
	case "stashes":
		return "Stashes to drop"
	case "worktrees":
		return "Worktrees to prune"
	default:
		return phase
	}
}

func phaseVerb(phase string) string {
	switch phase {
	case "stashes":
		return "Drop"
	case "worktrees":
		return "Prune"
	default:
		return "Delete"
	}
}

func phasePastVerb(phase string) string {
	switch phase {
	case "stashes":
		return "Dropped"
	case "worktrees":
		return "Pruned"
	default:
		return "Deleted"
	}
}

func phaseNothingMessage(phase string) string {
	return "Nothing " + strings.ToLower(phasePastVerb(phase)) + "."
}

func phaseCount(phase string, n int) string {
	var one, many string
	switch phase {
	case "local":
		one, many = "local branch", "local branches"
	case "remote":
		one, many = "branch on "+config.DefaultUpstreamRemote, "branches on "+config.DefaultUpstreamRemote
	case "stashes":
		one, many = "stash", "stashes"
	case "worktrees":
		one, many = "worktree", "worktrees"
	default:
		one, many = phase, phase
	}
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}

func emptyPhaseMessage(phase, defaultBranch string) string {
	switch phase {
	case "local":
		return "No local branches are merged into " + defaultBranch + " or a protected branch, or have a gone upstream."
	case "remote":
		if !state.AreThereRemotes() {
			return "No remotes; skipping " + config.DefaultUpstreamRemote + " branches."
		}
		return "No " + config.DefaultUpstreamRemote + " branches are merged into " + defaultBranch + " or a protected branch."
	case "stashes":
		return "There are no stashes."
	case "worktrees":
		return "No worktrees are missing their directory."
	default:
		return "Nothing to prune."
	}
}

func selectedDefaults(phase string, choices []prompt.Choice) []string {
	if phase == "stashes" {
		return nil
	}
	out := make([]string, len(choices))
	for i, c := range choices {
		out[i] = c.Value
	}
	return out
}

func listStashes(w io.Writer) error {
	text.CommandText(phaseHeader("stashes", ""))
	progress := newPruneProgress(w, phaseNoun("stashes"))
	choices, _, err := collectStashes(progress)
	progress.close()
	if err != nil {
		return err
	}
	if len(choices) == 0 {
		text.InfoText(emptyPhaseMessage("stashes", ""))
	}
	for _, c := range choices {
		line := c.Display
		if c.Description != "" {
			line += "  " + c.Description
		}
		text.InfoText(line)
	}
	text.InfoText(stashDropGuidance)
	return nil
}

func collectPhase(phase, defaultBranch string, progress *pruneProgress) ([]prompt.Choice, map[string]string, error) {
	switch phase {
	case "local":
		choices, err := collectLocal(defaultBranch, progress)
		return choices, nil, err
	case "remote":
		choices, err := collectRemote(defaultBranch, progress)
		return choices, nil, err
	case "stashes":
		return collectStashes(progress)
	case "worktrees":
		choices, err := collectWorktrees(progress)
		return choices, nil, err
	default:
		return nil, nil, fmt.Errorf("unknown prune phase %q", phase)
	}
}

type mergeTarget struct {
	name string
	ref  string
}

func mergeTargets(defaultBranch string) []mergeTarget {
	var out []mergeTarget
	seen := map[string]bool{}
	add := func(name, ref string) {
		if name == "" || ref == "" || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, mergeTarget{name: name, ref: ref})
	}
	add(defaultBranch, defaultBranch)
	remote := config.DefaultUpstreamRemote
	for _, branch := range strings.Fields(config.ProtectedBranches()) {
		if seen[branch] {
			continue
		}
		if tracking := remote + "/" + branch; state.IsRemoteBranch(tracking) {
			add(branch, tracking)
			continue
		}
		if state.LocalBranchExists(branch) {
			add(branch, branch)
		}
	}
	return out
}

func mergedInto(refsPrefix string, targets []mergeTarget) map[string]string {
	into := map[string]string{}
	for _, target := range targets {
		out := git.OutputOK("for-each-ref", "--merged="+target.ref, "--format=%(refname:short)", refsPrefix)
		for _, name := range splitLines(out) {
			if name == "" {
				continue
			}
			if _, ok := into[name]; ok {
				continue
			}
			into[name] = target.name
		}
	}
	return into
}

func collectLocal(defaultBranch string, progress *pruneProgress) ([]prompt.Choice, error) {
	out := git.OutputOK("for-each-ref", "--sort=committerdate", "--format", pruneRefFormat, "refs/heads")
	lines := splitLines(out)
	progress.setTotal(len(lines))
	merged := mergedInto("refs/heads", mergeTargets(defaultBranch))
	var choices []prompt.Choice
	for _, line := range lines {
		name, unix, short, upstream := splitRef(line)
		progress.tick(name)
		if name == "" || name == defaultBranch || config.IsBranchProtected(name) {
			continue
		}
		reason, ok := localPruneReason(name, upstream, merged)
		if !ok {
			continue
		}
		choices = append(choices, datedChoice(name, unix, short, reason))
	}
	return choices, nil
}

func localPruneReason(branch, upstreamShort string, merged map[string]string) (string, bool) {
	merge, _ := git.Output("config", "--get", "branch."+branch+".merge")
	merge = strings.TrimSpace(merge)
	if merge != "" {
		if state.IsThereUpstreamFor(branch) {
			return "", false
		}
		name := strings.TrimSpace(upstreamShort)
		if name == "" {
			name = strings.TrimPrefix(merge, "refs/remotes/")
			name = strings.TrimPrefix(name, "refs/heads/")
		}
		return "upstream " + name + " is gone", true
	}
	if target, ok := merged[branch]; ok {
		return "merged into " + target, true
	}
	return "", false
}

func collectRemote(defaultBranch string, progress *pruneProgress) ([]prompt.Choice, error) {
	if !state.AreThereRemotes() {
		return nil, nil
	}
	remote := config.DefaultUpstreamRemote
	out := git.OutputOK("for-each-ref", "--sort=committerdate", "--format", pruneRefFormat, "refs/remotes/"+remote)
	prefix := remote + "/"
	lines := splitLines(out)
	progress.setTotal(len(lines))
	merged := mergedInto("refs/remotes/"+remote, mergeTargets(defaultBranch))
	var choices []prompt.Choice
	for _, line := range lines {
		name, unix, short, _ := splitRef(line)
		progress.tick(name)
		if name == "" || !strings.HasPrefix(name, prefix) {
			continue
		}
		branch := strings.TrimPrefix(name, prefix)
		if branch == "" || branch == "HEAD" || branch == defaultBranch || config.IsBranchProtected(branch) {
			continue
		}
		target, ok := merged[name]
		if !ok {
			continue
		}
		choices = append(choices, datedChoice(name, unix, short, "merged into "+target))
	}
	return choices, nil
}

func collectStashes(progress *pruneProgress) ([]prompt.Choice, map[string]string, error) {
	common, err := repo.GitCommonDir()
	if err != nil {
		return nil, nil, err
	}
	entries, err := readStashes()
	if err != nil {
		return nil, nil, err
	}
	progress.setTotal(len(entries))
	var choices []prompt.Choice
	listed := map[string]string{}
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		progress.tick(entry.ref)
		listed[entry.ref] = entry.oid
		protected, err := recovery.Protects(common, entry.oid, entry.subject)
		if err != nil {
			return nil, nil, err
		}
		if protected {
			continue
		}
		choices = append(choices, datedChoice(entry.ref, entry.unix, entry.short, strings.ReplaceAll(entry.subject, "\t", " ")))
	}
	return choices, listed, nil
}

type pruneWorktree struct {
	path     string
	prunable bool
	locked   bool
}

func collectWorktrees(progress *pruneProgress) ([]prompt.Choice, error) {
	var choices []prompt.Choice
	worktrees := parseWorktrees(git.OutputOK("worktree", "list", "--porcelain"))
	progress.setTotal(len(worktrees))
	for _, wt := range worktrees {
		progress.tick(wt.path)
		if !wt.prunable || wt.locked || wt.path == "" {
			continue
		}
		choices = append(choices, prompt.Choice{Value: wt.path})
	}
	return choices, nil
}

func parseWorktrees(out string) []pruneWorktree {
	var recs []pruneWorktree
	var cur *pruneWorktree
	flush := func() {
		if cur != nil && cur.path != "" {
			recs = append(recs, *cur)
		}
		cur = nil
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		if cur == nil {
			cur = &pruneWorktree{}
		}
		if path, ok := strings.CutPrefix(line, "worktree "); ok {
			cur.path = path
			continue
		}
		if strings.HasPrefix(line, "prunable") {
			cur.prunable = true
		}
		if strings.HasPrefix(line, "locked") {
			cur.locked = true
		}
	}
	flush()
	return recs
}

func deletePhase(phase string, choices []prompt.Choice, selected []string, stashOIDs map[string]string) error {
	if len(selected) == 0 {
		return nil
	}
	switch phase {
	case "local":
		return deleteLocal(selected)
	case "remote":
		return deleteRemote(selected)
	case "worktrees":
		return deleteWorktrees(choices, selected)
	default:
		return fmt.Errorf("unknown prune phase %q", phase)
	}
}

func deleteLocal(names []string) error {
	for _, branch := range names {
		if err := git.Verbose("branch", "--delete", "--force", branch); err != nil {
			return err
		}
		if err := repo.ClearBranchSourceFromCWD(branch); err != nil {
			return err
		}
	}
	return nil
}

func deleteRemote(refs []string) error {
	if len(refs) == 0 {
		return nil
	}
	names := make([]string, len(refs))
	for i, ref := range refs {
		names[i] = strings.TrimPrefix(ref, config.DefaultUpstreamRemote+"/")
	}
	args := append([]string{"push", config.DefaultUpstreamRemote, "--delete"}, names...)
	return git.Verbose(args...)
}

type stashEntry struct {
	ref     string
	unix    string
	short   string
	subject string
	oid     string
}

func readStashes() ([]stashEntry, error) {
	out, err := git.Output("stash", "list", "--format="+pruneStashFormat)
	if err != nil {
		return nil, err
	}
	var entries []stashEntry
	for _, line := range splitLines(out) {
		ref, unix, short, subject, oid, ok := splitStash(line)
		if !ok {
			return nil, fmt.Errorf("stash list entry %q cannot be identified", line)
		}
		entries = append(entries, stashEntry{ref: ref, unix: unix, short: short, subject: subject, oid: oid})
	}
	return entries, nil
}

func deleteWorktrees(choices []prompt.Choice, selected []string) error {
	if len(selected) == 0 {
		return nil
	}
	sel := map[string]bool{}
	for _, name := range selected {
		sel[name] = true
	}
	var hold []string
	for _, c := range choices {
		if !sel[c.Value] {
			hold = append(hold, c.Value)
		}
	}
	locked := make([]string, 0, len(hold))
	for _, path := range hold {
		if err := git.Verbose("worktree", "lock", path); err != nil {
			_ = unlockWorktrees(locked)
			return err
		}
		locked = append(locked, path)
	}
	err := git.Verbose("worktree", "prune")
	uerr := unlockWorktrees(locked)
	if err != nil {
		return err
	}
	return uerr
}

func unlockWorktrees(paths []string) error {
	var first error
	for _, path := range paths {
		if err := git.Verbose("worktree", "unlock", path); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func splitLines(out string) []string {
	out = strings.TrimSpace(out)
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

func splitStash(line string) (ref, unix, short, subject, oid string, ok bool) {
	parts := strings.SplitN(line, "\t", 5)
	if len(parts) < 5 {
		return "", "", "", "", "", false
	}
	oid = strings.TrimSpace(parts[0])
	ref = strings.TrimSpace(parts[1])
	unix = strings.TrimSpace(parts[2])
	short = strings.TrimSpace(parts[3])
	subject = strings.TrimSpace(parts[4])
	if !stashOID(oid) || ref == "" {
		return "", "", "", "", "", false
	}
	return ref, unix, short, subject, oid, true
}

func stashOID(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, c := range s {
		if c < '0' || (c > '9' && c < 'a') || c > 'f' {
			return false
		}
	}
	return true
}

func splitRef(line string) (name, unix, short, extra string) {
	parts := strings.SplitN(line, "\t", 4)
	if len(parts) > 0 {
		name = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		unix = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 {
		short = strings.TrimSpace(parts[2])
	}
	if len(parts) > 3 {
		extra = strings.TrimSpace(parts[3])
	}
	return name, unix, short, extra
}

func datedChoice(value, unix, short, desc string) prompt.Choice {
	return prompt.Choice{
		Value:       value,
		Display:     dateColumn(unix, short) + "  " + trimName(value, pruneNameMaxLen),
		Description: desc,
	}
}

func dateColumn(unix, short string) string {
	rel := strings.Repeat(" ", len("[NN u ago]"))
	if n, err := strconv.ParseInt(unix, 10, 64); err == nil {
		rel = statefmt.RelTime(n, pruneNowUnix())
	}
	return rel + "  " + fmt.Sprintf("%-10s", short)
}

func trimName(name string, max int) string {
	if len(name) <= max {
		return name
	}
	return name[:max-3] + "..."
}
