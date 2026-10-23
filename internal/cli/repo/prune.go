package repo

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/statefmt"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var pruneID = cmdid.ID{Command: "repo", Action: "prune"}

const (
	pruneRefFormat   = "%(refname:short)%09%(committerdate:unix)%09%(committerdate:short)%09%(upstream:short)"
	pruneStashFormat = "%gd%x09%ct%x09%cs%x09%s"
	pruneNameMaxLen  = 40
)

var pruneNowUnix = func() int64 { return time.Now().Unix() }

var prunePhaseNames = []string{"local", "remote", "stashes", "worktrees"}

func newPruneCommand() *cobra.Command {
	c := &cobra.Command{
		Use:       "prune [phase...]",
		Short:     "Cleans up branches, stashes, and worktrees",
		Long:      "Deletes local branches merged into the default development branch or a protected branch, or whose upstream is gone; deletes branches on origin merged into those same branches; drops stashes you pick; and prunes worktrees whose directory is gone. Pass phase names to choose. Interactive mode asks which phases to run, lets you pick candidates, and confirms before each phase applies; non-interactive mode with no phases runs local only and applies without asking.",
		ValidArgs: append([]string(nil), prunePhaseNames...),
		Args:      cobra.OnlyValidArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, pruneID, func() error {
				return pruneRun(cmd, args)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func pruneRun(cmd *cobra.Command, args []string) error {
	p := prompt.FromContext(cmd.Context())
	phases := orderedPhases(args)
	if prompt.NonInteractive(p) && containsPhase(phases, "stashes") {
		return fmt.Errorf("stashes require interactive mode")
	}
	if err := prunePrepare(); err != nil {
		return err
	}
	defaultBranch := config.DefaultBranch()
	if len(args) == 0 {
		if prompt.NonInteractive(p) {
			phases = []string{"local"}
		} else {
			var err error
			phases, err = prompt.PickMany(p, "What to clean up", phaseChoices(), append([]string(nil), prunePhaseNames...))
			if err != nil {
				return err
			}
			phases = orderedPhases(phases)
		}
	}
	for _, phase := range phases {
		if err := prunePhase(cmd.OutOrStdout(), p, phase, defaultBranch); err != nil {
			return err
		}
	}
	return nil
}

func prunePrepare() error {
	defaultBranch := config.DefaultBranch()
	if err := git.Verbose("checkout", defaultBranch); err != nil {
		return err
	}
	if state.IsThereUpstreamFor(defaultBranch) {
		if err := git.Verbose("fetch", "--all", "--prune"); err != nil {
			text.InfoText("As the remotes can't be fetched, the current local version is used.")
		} else {
			if err := git.Verbose("rebase"); err != nil {
				return err
			}
		}
	}
	return nil
}

func phaseChoices() []prompt.Choice {
	return []prompt.Choice{
		{Value: "local", Description: "Delete local branches that are merged or whose upstream is gone."},
		{Value: "remote", Description: "Delete remote branches that are merged."},
		{Value: "stashes", Description: "Drop stashes you pick; none are selected by default."},
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

func containsPhase(phases []string, name string) bool {
	for _, phase := range phases {
		if phase == name {
			return true
		}
	}
	return false
}

func prunePhase(w io.Writer, p prompt.Prompter, phase, defaultBranch string) error {
	text.CommandText(phaseHeader(phase, defaultBranch))
	progress := newPruneProgress(w, phaseNoun(phase))
	choices, err := collectPhase(phase, defaultBranch, progress)
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
	if err := deletePhase(phase, choices, selected); err != nil {
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
		return "Pruning stashes (only the ones you select are dropped)..."
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

func collectPhase(phase, defaultBranch string, progress *pruneProgress) ([]prompt.Choice, error) {
	switch phase {
	case "local":
		return collectLocal(defaultBranch, progress)
	case "remote":
		return collectRemote(defaultBranch, progress)
	case "stashes":
		return collectStashes(progress)
	case "worktrees":
		return collectWorktrees(progress)
	default:
		return nil, fmt.Errorf("unknown prune phase %q", phase)
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

func collectStashes(progress *pruneProgress) ([]prompt.Choice, error) {
	out := git.OutputOK("stash", "list", "--format="+pruneStashFormat)
	lines := splitLines(out)
	progress.setTotal(len(lines))
	var choices []prompt.Choice
	for i := len(lines) - 1; i >= 0; i-- {
		ref, unix, short, subject := splitRef(lines[i])
		progress.tick(ref)
		if ref == "" {
			continue
		}
		choices = append(choices, datedChoice(ref, unix, short, subject))
	}
	return choices, nil
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

func deletePhase(phase string, choices []prompt.Choice, selected []string) error {
	if len(selected) == 0 {
		return nil
	}
	switch phase {
	case "local":
		return deleteLocal(selected)
	case "remote":
		return deleteRemote(selected)
	case "stashes":
		return deleteStashes(selected)
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

func deleteStashes(refs []string) error {
	sorted := append([]string(nil), refs...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return stashIndex(sorted[i]) > stashIndex(sorted[j])
	})
	for _, ref := range sorted {
		if err := git.Verbose("stash", "drop", ref); err != nil {
			return err
		}
	}
	return nil
}

func stashIndex(ref string) int {
	start := strings.LastIndex(ref, "{")
	end := strings.LastIndex(ref, "}")
	if start < 0 || end <= start+1 {
		return -1
	}
	n, err := strconv.Atoi(ref[start+1 : end])
	if err != nil {
		return -1
	}
	return n
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
