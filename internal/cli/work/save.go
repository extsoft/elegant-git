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
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
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

Optional target is new (a new commit), HEAD, or a unique commit hash (short or full). If omitted, interactive mode asks when the branch has unique commits; non-interactive mode creates a new commit. The last unique commit is amended; an older one is applied as a fixup and autosquashed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, saveID, func() error {
				return saveRun(cmd, args, spec)
			})
		},
	}
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
	branch := cliruntime.CurrentBranch()
	if config.IsBranchProtected(branch) {
		if cliruntime.StdinIsInteractive(cmd.Context()) {
			return saveViaStartOnProtected(cmd, branch, args, spec)
		}
		cliruntime.ExitProtectedNoCommits(branch)
	}
	return saveCommit(cmd, args, spec)
}

func saveViaStartOnProtected(cmd *cobra.Command, branch string, args []string, spec argspec.Spec) error {
	text.InfoBox(fmt.Sprintf("Warning: no direct commits on the protected '%s' branch.", branch))
	text.InfoText("Starting `eg work start` to create a feature branch first.")
	var name, fromRef string
	start := startSpec(&name, &fromRef)
	if err := cliruntime.RunWithWorkflows(cmd, startID, func() error {
		return startRun(cmd, nil, start)
	}); err != nil {
		return err
	}
	return saveCommit(cmd, args, spec)
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
	if err := applySaveTarget(cmd, target); err != nil {
		return err
	}
	return offerPushAfterSave(cmd, branch)
}

func offerPushAfterSave(cmd *cobra.Command, branch string) error {
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
		text.ErrorText("Saved, but push failed: " + err.Error())
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
	return hash, formatSaveRelTime(unix, saveNowUnix()), subject
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

func formatSaveRelTime(unix, now int64) string {
	sec := now - unix
	if sec < 0 {
		sec = 0
	}
	var n int64
	var unit byte
	switch {
	case sec < 60:
		n, unit = sec, 's'
	case sec < 60*60:
		n, unit = sec/60, 'm'
	case sec < 24*60*60:
		n, unit = sec/3600, 'h'
	case sec < 100*24*60*60:
		n, unit = sec/(24*60*60), 'd'
	case sec < 100*7*24*60*60:
		n, unit = sec/(7*24*60*60), 'w'
	default:
		n, unit = sec/(365*24*60*60), 'y'
	}
	if n > 99 {
		n = 99
	}
	return fmt.Sprintf("[%2d %c ago]", n, unit)
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

func applySaveTarget(cmd *cobra.Command, target string) error {
	if target == "" || target == saveTargetNew {
		return git.Verbose("commit")
	}
	head, err := revParse("HEAD")
	if err != nil {
		return err
	}
	picked, err := revParse(target)
	if err != nil {
		return err
	}
	if head == picked {
		return cliruntime.RunWithWorkflows(cmd, amendID, func() error {
			return git.Verbose("commit", "--amend")
		})
	}
	return applyFixup(target)
}

func applyFixup(target string) error {
	orig, err := revParse("HEAD")
	if err != nil {
		return err
	}
	parent, err := revParse(target + "^")
	if err != nil {
		return fmt.Errorf("cannot find parent of %s: %w", target, err)
	}
	if err := git.Verbose("commit", "--fixup="+target); err != nil {
		return err
	}
	err = git.Verbose("-c", "sequence.editor=true", "rebase", "--interactive", "--autosquash", "--autostash", parent)
	if err == nil {
		return nil
	}
	_ = git.Verbose("rebase", "--abort")
	_ = git.Verbose("reset", "--soft", orig)
	return err
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
