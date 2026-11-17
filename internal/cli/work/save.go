package work

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/statefmt"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/pipe"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var saveID = cmdid.ID{Command: "work", Action: "save"}

const saveTargetNew = "new"
const saveTargetHEAD = "HEAD"
const pushAfterSaveDifferent = "__push_different__"
const pushAfterSaveSkip = "__push_skip__"
const saveCommitLogFormat = "%h%x09%at%x09%s"

var saveNowUnix = func() int64 { return time.Now().Unix() }

type saveCommitRow struct {
	hash, rel, subject string
}

func newSaveCommand() *cobra.Command {
	var target string
	spec := saveSpec(&target)
	c := &cobra.Command{
		Use:   "save [target]",
		Short: "Commits current modifications",
		Long: `Commits current modifications, or folds them into a unique commit already on the branch.

Optional target is new (a new commit), HEAD, or a unique commit hash (short or full). If omitted, interactive mode asks when the branch has unique commits; non-interactive mode creates a new commit. The last unique commit is amended; an older one is applied as a fixup and autosquashed.

The commit consumes the index you stage with git add --interactive. That staging session is Git's, including when this command is non-interactive. --dirty does not choose those commit inputs and does not discard them. It applies only to edits left after an older-target commit, and to a nested work start or work push. carry stashes those leftovers and restores them after the autosquash. stash leaves them in the stash. discard reverts modifications and deletions of committed leftovers and leaves staged additions and rename targets as untracked files. cancel before the fixup commit changes nothing. A refusal after the commit reports the commit and leaves it in place. A non-interactive run with leftovers fails unless --dirty is set. Nested start and push reuse that choice. Stash on a protected branch stashes those inputs, names the stash, and stops with a workflow error. Save does not commit them. Discard there is refused.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, saveID, func() error {
				return saveRun(cmd, args, spec)
			})
		},
	}
	AddDirtyFlags(c)
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func saveSpec(target *string) argspec.Spec {
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInputWithComplete("target", 0, false, "How to save", target, nil, saveTargetComplete, false),
	}}
}

func saveTargetComplete(_ context.Context) ([]argspec.Choice, error) {
	rows := saveUniqueCommits(cliruntime.CurrentBranch())
	out := make([]argspec.Choice, 0, len(rows)+2)
	out = append(out, argspec.Choice{Value: saveTargetNew, Description: "Create a new commit."})
	if len(rows) > 0 {
		out = append(out, argspec.Choice{Value: saveTargetHEAD, Description: "The last unique commit (amend)."})
	}
	for _, row := range rows {
		out = append(out, argspec.Choice{Value: row.hash, Description: row.subject})
	}
	return out, nil
}

func saveRun(cmd *cobra.Command, args []string, spec argspec.Spec) error {
	if _, _, _, err := readDirtyFlags(cmd); err != nil {
		return err
	}
	branch := cliruntime.CurrentBranch()
	if config.IsBranchProtected(branch) {
		if saveInteractive(cmd) {
			return saveViaStartOnProtected(cmd, branch, args, spec)
		}
		cliruntime.ExitProtectedNoCommits(branch)
	}
	return saveCommit(cmd, args, spec)
}

func saveInteractive(cmd *cobra.Command) bool {
	ctx := context.Background()
	if cmd != nil && cmd.Context() != nil {
		ctx = cmd.Context()
	}
	if !cliruntime.StdinIsInteractive(ctx) {
		return false
	}
	return !prompt.NonInteractive(prompt.FromContext(ctx))
}

func saveViaStartOnProtected(cmd *cobra.Command, branch string, args []string, spec argspec.Spec) error {
	text.InfoBox(fmt.Sprintf("Warning: no direct commits on the protected '%s' branch.", branch))
	text.InfoText("Starting `eg work start` to create a feature branch first.")
	var name, fromRef string
	start := startSpec(&name, &fromRef)
	chosen := ""
	ctx := withChoiceGuard(cmd.Context(), func(choice string) error {
		chosen = choice
		if choice == "discard" {
			return workflowText("Save will not discard the changes it is about to commit. Carry them onto the new branch, stash them and stop, or cancel.")
		}
		return nil
	})
	ctx, stashed := withRecordedStash(ctx)
	cmd.SetContext(ctx)
	if err := cliruntime.RunWithWorkflows(cmd, startID, func() error {
		return startRun(cmd, nil, start)
	}); err != nil {
		return err
	}
	if chosen != "" && cmd.Flags().Lookup("dirty") != nil {
		_ = cmd.Flags().Set("dirty", chosen)
	}
	if chosen == "stash" {
		if stashed.kept.OID == "" {
			return fmt.Errorf("stash handoff did not report a reachable stash. Inspect it with `eg repo doctor`")
		}
		return workflowText(stoppedSaveMessage(stashed.kept))
	}
	return saveCommit(cmd, args, spec)
}

func stoppedSaveMessage(kept pipe.Retention) string {
	where := kept.OID
	if kept.Selector != "" {
		where = kept.Selector + " " + kept.OID
	}
	return fmt.Sprintf("Those changes stay in the stash as %s. Save did not commit them. Restore them with `git stash apply --index %s`.", where, kept.OID)
}

func saveCommit(cmd *cobra.Command, args []string, spec argspec.Spec) error {
	if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
		return err
	}
	branch := cliruntime.CurrentBranch()
	target := strings.TrimSpace(spec.Inputs[0].Get())
	var err error
	if target == "" {
		target, err = pickSaveTarget(cmd, branch)
	} else {
		target, err = resolveSaveTarget(branch, target)
	}
	if err != nil {
		return err
	}
	if err := git.Verbose("add", "--interactive"); err != nil {
		return err
	}
	saved, err := applySaveTarget(cmd, target)
	if err != nil {
		return err
	}
	return offerPushAfterSave(cmd, branch, saved)
}

func offerPushAfterSave(cmd *cobra.Command, branch, saved string) error {
	if branch == "HEAD" || config.IsBranchProtected(branch) || !state.AreThereRemotes() {
		return nil
	}
	p := prompt.FromContext(cmd.Context())
	if prompt.NonInteractive(p) {
		return nil
	}
	defaultBranch := pushRemoteBranch(branch, "", cliruntime.BranchUpstreamShort(branch))
	choices := pushAfterSaveChoices(defaultBranch)
	ans, err := p.Pick("Push?", choices, defaultBranch)
	if err != nil {
		if errors.Is(err, prompt.ErrUserCancelled) {
			return nil
		}
		return err
	}
	if ans == pushAfterSaveSkip {
		return nil
	}
	remoteBranch := ans
	if ans == pushAfterSaveDifferent {
		remoteBranch, err = pickPushRemoteBranchName(p)
		if err != nil {
			if errors.Is(err, prompt.ErrUserCancelled) {
				return nil
			}
			return err
		}
	}
	if err := cliruntime.RunWithWorkflows(cmd, pushID, func() error {
		return pushRun(cmd, []string{remoteBranch})
	}); err != nil {
		if saved == "" {
			text.ErrorText("Saved, but push failed: " + err.Error())
		} else {
			text.ErrorText(fmt.Sprintf("Saved %s, but push failed: %s", saved, err.Error()))
		}
		return nil
	}
	return nil
}

func pushAfterSaveChoices(defaultBranch string) []prompt.Choice {
	return []prompt.Choice{
		{Value: defaultBranch, Description: "Publish to this branch."},
		{Value: pushAfterSaveDifferent, Display: "different", Description: "Publish to a different branch name."},
		{Value: pushAfterSaveSkip, Display: "no", Description: "Do not push."},
	}
}

func pickPushRemoteBranchName(p prompt.Prompter) (string, error) {
	for {
		name, err := p.String("Branch name", "")
		if err != nil {
			return "", err
		}
		name = strings.TrimSpace(name)
		if name == "" {
			text.ErrorText("Branch name is required.")
			continue
		}
		if err := git.CheckBranchName(name); err != nil {
			text.ErrorText(fmt.Sprintf("Invalid branch name %q: %s", name, err.Error()))
			continue
		}
		return name, nil
	}
}

func pickSaveTarget(cmd *cobra.Command, branch string) (string, error) {
	p := prompt.FromContext(cmd.Context())
	if prompt.NonInteractive(p) {
		return saveTargetNew, nil
	}
	choices := saveTargetChoices(branch)
	if len(choices) < 2 {
		return saveTargetNew, nil
	}
	ans, err := p.Pick("How to save", choices, saveTargetNew)
	if err != nil {
		return "", err
	}
	return saveRefFromChoice(ans), nil
}

func saveTargetChoices(branch string) []prompt.Choice {
	out := []prompt.Choice{{Value: saveTargetNew, Description: "Create a new commit."}}
	for _, row := range saveUniqueCommits(branch) {
		out = append(out, prompt.Choice{
			Value:       row.hash,
			Display:     saveChoiceValue(row.rel, row.hash),
			Description: row.subject,
		})
	}
	return out
}

func saveUniqueCommits(branch string) []saveCommitRow {
	latest := config.FreshestBranchSourceBranch(branch)
	log, err := git.Output("log", "--format="+saveCommitLogFormat, latest+".."+branch)
	if err != nil {
		return nil
	}
	var out []saveCommitRow
	for _, line := range strings.Split(log, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		hash, rel, subject := parseSaveCommitLogLine(line)
		if hash == "" {
			continue
		}
		out = append(out, saveCommitRow{hash: hash, rel: rel, subject: subject})
	}
	return out
}

func parseSaveCommitLogLine(line string) (hash, rel, subject string) {
	hash, rest, ok := strings.Cut(line, "\t")
	if !ok {
		return "", "", ""
	}
	at, subject, ok := strings.Cut(rest, "\t")
	if !ok {
		return hash, "", rest
	}
	unix, err := strconv.ParseInt(at, 10, 64)
	if err != nil {
		return hash, "", subject
	}
	return hash, statefmt.RelTime(unix, saveNowUnix()), subject
}

func saveChoiceValue(rel, hash string) string {
	if rel == "" {
		return hash
	}
	return rel + "  " + hash
}

func saveRefFromChoice(target string) string {
	target = strings.TrimSpace(target)
	if target == "" || target == saveTargetNew {
		return target
	}
	if _, hash, ok := strings.Cut(target, "]  "); ok && hash != "" {
		return hash
	}
	return target
}

func resolveSaveTarget(branch, target string) (string, error) {
	target = saveRefFromChoice(target)
	if target == "" || target == saveTargetNew {
		return saveTargetNew, nil
	}
	oid, err := revParse(target)
	if err != nil {
		return "", fmt.Errorf("target: %q is not a valid choice", target)
	}
	for _, row := range saveUniqueCommits(branch) {
		got, err := revParse(row.hash)
		if err != nil {
			continue
		}
		if got == oid {
			return row.hash, nil
		}
	}
	return "", fmt.Errorf("target: %q is not a valid choice", target)
}

func applySaveTarget(cmd *cobra.Command, target string) (string, error) {
	if target == "" || target == saveTargetNew {
		if err := git.Verbose("commit"); err != nil {
			return "", err
		}
		saved, _ := revParse("HEAD")
		return saved, nil
	}
	head, err := revParse("HEAD")
	if err != nil {
		return "", err
	}
	picked, err := revParse(target)
	if err != nil {
		return "", err
	}
	if head == picked {
		if err := cliruntime.RunWithWorkflows(cmd, amendID, func() error {
			return git.Verbose("commit", "--amend")
		}); err != nil {
			return "", err
		}
		saved, _ := revParse("HEAD")
		return saved, nil
	}
	return applyFixup(cmd, target)
}

func applyFixup(cmd *cobra.Command, target string) (string, error) {
	parent, err := revParse(target + "^")
	if err != nil {
		return "", fmt.Errorf("cannot find parent of %s: %w", target, err)
	}
	before, err := state.ReadSnapshot()
	if err != nil {
		return "", err
	}
	if before.Operation != "" || len(before.Unmerged) > 0 {
		return "", workflowText("Conflict resolutions are not stashed or discarded. Finish or abort the current operation first.")
	}
	explicit, _, _, err := readDirtyFlags(cmd)
	if err != nil {
		return "", err
	}
	choice := ""
	if hasLeftovers(before) {
		choice, err = chooseDirty(navRequest{cmd: cmd, ctx: cmd.Context(), verb: "work save", dest: "the autosquash"}, explicit)
		if err != nil {
			return "", err
		}
		if choice == "cancel" {
			return "", cancelled("work save")
		}
		if err := refusePredictableLeftovers(choice, before); err != nil {
			return "", err
		}
	}
	if err := git.Verbose("commit", "--fixup="+target); err != nil {
		return "", err
	}
	saved, err := revParse("HEAD")
	if err != nil {
		return "", err
	}
	if err := autosquash(cmd, parent, saved, choice, explicit); err != nil {
		return "", err
	}
	return saved, nil
}

func hasLeftovers(snap state.Snapshot) bool {
	return len(snap.Unstaged) > 0 || len(snap.Untracked) > 0
}

func refusePredictableLeftovers(choice string, snap state.Snapshot) error {
	if choice == "discard" {
		if len(snap.Unstaged) == 0 && len(snap.Unmerged) == 0 {
			return workflowText(untrackedDiscardMessage)
		}
		return nil
	}
	if err := refuseUnsupported(); err != nil {
		return err
	}
	return refusePreserve("", true)
}

func autosquash(cmd *cobra.Command, parent, saved, choice, explicit string) error {
	left := fmt.Sprintf("saved %s; autosquash did not run, and that commit was left in place", saved)
	after, err := state.ReadSnapshot()
	if err != nil {
		return annotate(err, left)
	}
	if after.Operation != "" || len(after.Unmerged) > 0 {
		return fmt.Errorf("%s", left)
	}
	if hasLeftovers(after) && choice == "" {
		choice, err = chooseDirty(navRequest{cmd: cmd, ctx: cmd.Context(), verb: "work save", dest: "the autosquash"}, explicit)
		if err != nil {
			return annotate(err, left)
		}
		if choice == "cancel" {
			return fmt.Errorf("%s", left)
		}
	}
	if choice != "" && cmd != nil && cmd.Flags().Lookup("dirty") != nil {
		_ = cmd.Flags().Set("dirty", choice)
	}
	rebase := func() error {
		return git.Verbose("-c", "sequence.editor=true", "rebase", "--interactive", "--autosquash", parent)
	}
	unfinished := fmt.Sprintf("saved %s; autosquash did not finish, and that commit was left in place", saved)
	if !hasLeftovers(after) {
		if err := rebase(); err != nil {
			return annotate(err, unfinished)
		}
		return nil
	}
	err = PrepareHistory(History{
		Cmd:   cmd,
		ID:    saveID,
		Verb:  "work save",
		Dest:  "the autosquash",
		Allow: func(state.Snapshot) bool { return false },
		Resolve: func() (string, error) {
			return parent, nil
		},
		Replay: true,
		Body:   rebase,
	})
	if err != nil {
		return annotate(err, unfinished)
	}
	return nil
}

func revParse(ref string) (string, error) {
	out, err := git.Output("rev-parse", ref)
	out = strings.TrimSpace(out)
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", fmt.Errorf("cannot resolve %s", ref)
	}
	return out, nil
}
