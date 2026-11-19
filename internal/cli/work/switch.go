package work

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/exitcode"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/pipe"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var switchID = cmdid.ID{Command: "work", Action: "switch"}

const otherRevisionChoice = "other revision"

func newSwitchCommand() *cobra.Command {
	spec := switchSpec()
	c := &cobra.Command{
		Use:   "switch [ref] [local-branch]",
		Short: "Switches to a branch, or creates one from a remote branch, tag, or revision.",
		Long: `Switches to a branch, or creates one from a remote branch, tag, or revision.

When there are uncommitted changes, choose carry, stash, discard, or cancel. carry moves tracked and non-ignored untracked files onto the branch you switch to and keeps the staged and unstaged split. stash leaves them in the stash instead. discard restores modifications and deletions of committed files. Without --untracked=delete it leaves staged additions, rename targets, and other non-ignored untracked files in place. --untracked=delete removes those listed new-path files permanently and keeps no backup. cancel stops before any of that. A non-interactive run with uncommitted changes fails unless --dirty is set. Switching to the current branch changes nothing. A clean tree does not need --dirty.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, switchID, func() error {
				return switchRun(cmd, args)
			})
		},
	}
	addDirtyFlags(c)
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func switchSpec() argspec.Spec {
	var ref, local string
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.GitBacked(argspec.PositionalInputWithComplete("ref", 0, true, "Branch, remote branch, tag, or revision", &ref, nil, sources.RefsListed, false)),
		argspec.PositionalInput("local-branch", 1, false, "Local branch name", &local, nil),
	}}
}

func switchRun(cmd *cobra.Command, args []string) error {
	ref := ""
	if len(args) >= 1 {
		ref = strings.TrimSpace(args[0])
	}
	localBranch := ""
	if len(args) >= 2 {
		localBranch = strings.TrimSpace(args[1])
	}
	if err := rejectLeadingDash(cmd, ref); err != nil {
		return err
	}
	if err := guardSwitch(); err != nil {
		return err
	}
	p := prompt.FromContext(cmd.Context())
	picked := false
	if ref == "" {
		if prompt.NonInteractive(p) {
			return &argspec.ErrMissingRequired{Names: []string{"ref"}}
		}
		var err error
		ref, err = pickSwitchRef(cmd.Context(), p)
		if err != nil {
			return err
		}
		picked = true
		if err := rejectLeadingDash(cmd, ref); err != nil {
			return err
		}
	}
	return switchTo(cmd, p, ref, localBranch, picked)
}

func rejectLeadingDash(cmd *cobra.Command, ref string) error {
	if strings.HasPrefix(ref, "-") {
		return cliruntime.NewUsageError(cmd, fmt.Errorf("ref must not start with '-'"))
	}
	return nil
}

func guardSwitch() error {
	if op := state.InProgressOperation(); op != "" {
		return operationError(op)
	}
	if state.IsBareRepository() {
		return workflowText("This is a bare repository; it has no working tree.")
	}
	return nil
}

func operationError(op string) error {
	var msg string
	switch op {
	case "merge":
		msg = "A merge is in progress. Finish it with 'git commit' or abort it with 'git merge --abort'."
	case "cherry-pick":
		msg = "A cherry-pick is in progress. Finish it with 'git cherry-pick --continue' or abort it with 'git cherry-pick --abort'."
	case "revert":
		msg = "A revert is in progress. Finish it with 'git revert --continue' or abort it with 'git revert --abort'."
	case "rebase":
		msg = "A rebase is in progress. Finish it with 'git rebase --continue' or abort it with 'git rebase --abort'."
	case "bisect":
		msg = "A bisect is in progress. Finish it with 'git bisect good' or 'git bisect bad', or abort it with 'git bisect reset'."
	default:
		msg = fmt.Sprintf("A %s is in progress.", op)
	}
	return workflowText(msg)
}

func pickSwitchRef(ctx context.Context, p prompt.Prompter) (string, error) {
	choices, err := sources.Refs(ctx)
	if err != nil {
		return "", err
	}
	promptChoices := make([]prompt.Choice, 0, len(choices)+1)
	for _, c := range choices {
		promptChoices = append(promptChoices, prompt.Choice{Value: c.Value, Description: c.Description})
	}
	promptChoices = append(promptChoices, prompt.Choice{
		Value:       otherRevisionChoice,
		Description: "Type a commit or other revision.",
	})
	val, err := p.Pick("Branch, remote branch, or tag", promptChoices, "")
	if err != nil {
		return "", err
	}
	val = strings.TrimSpace(val)
	if i := strings.IndexByte(val, '\t'); i >= 0 {
		val = val[:i]
	}
	if val == otherRevisionChoice {
		line, err := prompt.ValidatedLine(p, "Revision", "", func(s string) string {
			if strings.TrimSpace(s) == "" {
				return "revision is required"
			}
			return ""
		})
		if err != nil {
			return "", err
		}
		val = strings.TrimSpace(line)
	}
	if val == "" {
		return "", fmt.Errorf("ref is required")
	}
	return val, nil
}

func switchTo(cmd *cobra.Command, p prompt.Prompter, ref, localBranch string, picked bool) error {
	if !picked && !isExactLocalInput(ref) && state.AreThereRemotes() {
		cliruntime.FetchOrInform()
	}
	resolved, err := resolveSwitchRef(ref)
	if err != nil {
		return err
	}
	if resolved.kind == switchLocal {
		if localBranch != "" {
			return cliruntime.NewUsageError(cmd, fmt.Errorf("local-branch does not apply to a local branch"))
		}
		if cliruntime.CurrentBranch() == resolved.name {
			text.InfoText(fmt.Sprintf("Already on '%s'", resolved.name))
			return nil
		}
	}
	if resolved.kind == switchTag || resolved.kind == switchRev {
		return switchNonBranch(cmd, p, resolved, localBranch)
	}
	diffRef := resolved.name
	if resolved.kind == switchLocal {
		diffRef = "refs/heads/" + resolved.name
	}
	return withSwitchChanges(cmd, resolved.name, diffRef, func() error {
		return switchAct(p, resolved, localBranch)
	})
}

func switchAct(p prompt.Prompter, resolved switchResolved, localBranch string) error {
	switch resolved.kind {
	case switchLocal:
		if err := git.Verbose("checkout", resolved.name, "--"); err != nil {
			return err
		}
		return pipe.RememberRestoreBranch(resolved.name)
	case switchRemote:
		local, fold, err := resolveLocalBranch(p, resolved.name, localBranch)
		if err != nil {
			return err
		}
		checkedOut, err := checkoutTrackedLocal(p, local, resolved.name, fold, trackDivergenceStay)
		if err != nil {
			return err
		}
		return pipe.RememberRestoreBranch(checkedOut)
	default:
		return notARef(resolved.input)
	}
}

func switchNonBranch(cmd *cobra.Command, p prompt.Prompter, resolved switchResolved, localBranch string) error {
	if resolved.commit == "" {
		return notARef(resolved.input)
	}
	name, err := resolveNonBranchName(p, resolved, localBranch)
	if err != nil {
		return err
	}
	return withSwitchChanges(cmd, resolved.commit, resolved.commit, func() error {
		if err := git.Verbose("checkout", "-b", name, "--no-track", resolved.commit); err != nil {
			return err
		}
		if err := pipe.RememberRestoreBranch(name); err != nil {
			return err
		}
		return config.SetBranchSourceBranch(name, resolved.input)
	})
}

func resolveNonBranchName(p prompt.Prompter, resolved switchResolved, localBranch string) (string, error) {
	suggested, err := autoBranchBase(resolved)
	if err != nil {
		return "", err
	}
	branches, err := state.LocalBranchNames()
	if err != nil {
		return "", err
	}
	if prompt.NonInteractive(p) {
		name := suggested
		if strings.TrimSpace(localBranch) != "" {
			name = strings.TrimSpace(localBranch)
		}
		if problem := submitNewBranchNameProblem(name, branches); problem != "" {
			return "", workflowText(problem)
		}
		return name, nil
	}
	initial := suggested
	if strings.TrimSpace(localBranch) != "" {
		initial = strings.TrimSpace(localBranch)
	}
	fold := prepareCaseFold()
	val, err := prompt.ValidatedLine(p, "Local branch name", initial, func(s string) string {
		return liveBranchProblem(strings.TrimSpace(s), branches, fold)
	})
	if err != nil {
		if errors.Is(err, io.EOF) {
			return "", errors.New("end of input")
		}
		return "", err
	}
	name := strings.TrimSpace(val)
	if problem := submitNewBranchNameProblem(name, branches); problem != "" {
		return resolveNewBranchNameFold(p, name, branches, name, &fold)
	}
	return name, nil
}

func autoBranchBase(resolved switchResolved) (string, error) {
	if resolved.kind == switchTag {
		return "tag-" + resolved.name, nil
	}
	short, err := shortCommit(resolved.commit)
	if err != nil {
		return "", err
	}
	return "rev-" + short, nil
}

func shortCommit(sha string) (string, error) {
	out, err := git.Output("rev-parse", "--short", "--verify", "--end-of-options", sha)
	if err != nil {
		return "", git.ErrorFromOutput(out, err)
	}
	short := strings.TrimSpace(out)
	if short == "" {
		return "", fmt.Errorf("empty short commit")
	}
	return short, nil
}

func peelCommit(ref string) (string, bool) {
	out, err := git.Output("rev-parse", "--verify", "--quiet", "--end-of-options", ref+"^{commit}")
	if err != nil {
		return "", false
	}
	sha := strings.TrimSpace(out)
	if sha == "" {
		return "", false
	}
	return sha, true
}

func resolveTag(name, input string) (switchResolved, error) {
	commit, ok := peelCommit("refs/tags/" + name)
	if !ok {
		return switchResolved{}, notARef(input)
	}
	return switchResolved{kind: switchTag, name: name, input: input, commit: commit}, nil
}

func withSwitchChanges(cmd *cobra.Command, dest, diffRef string, logic func() error) error {
	return applyNavigation(navRequest{
		cmd:  cmd,
		ctx:  cmd.Context(),
		id:   switchID,
		verb: "work switch",
		dest: dest,
		target: func() (string, error) {
			return diffRef, nil
		},
		body: logic,
	})
}

type switchKind int

const (
	switchLocal switchKind = iota
	switchRemote
	switchTag
	switchRev
)

type switchResolved struct {
	kind   switchKind
	name   string
	input  string
	commit string
}

type switchHit struct {
	kind switchKind
	name string
}

func resolveSwitchRef(input string) (switchResolved, error) {
	heads := refList("refs/heads")
	remotes := remoteList()
	tags := refList("refs/tags")
	if kind, name, ok := qualifiedKind(input); ok {
		return resolveQualified(input, kind, name, heads, remotes, tags)
	}
	headSet := refSet(heads)
	remoteSet := refSet(remotes)
	tagSet := refSet(tags)
	if headSet[input] {
		noteOtherKinds(input, switchLocal, remoteSet, tagSet)
		return switchResolved{switchLocal, input, input, ""}, nil
	}
	if remoteSet[input] {
		noteOtherKinds(input, switchRemote, remoteSet, tagSet)
		return switchResolved{switchRemote, input, input, ""}, nil
	}
	if matches := remoteBranchesNamed(input, remotes); len(matches) == 1 {
		noteOtherKinds(input, switchRemote, remoteSet, tagSet)
		return switchResolved{switchRemote, matches[0], input, ""}, nil
	} else if len(matches) > 1 {
		return switchResolved{}, branchListError(fmt.Sprintf("'%s' matches more than one remote branch:", input), matches)
	}
	if tagSet[input] {
		return resolveTag(input, input)
	}
	if commit, ok := peelCommit(input); ok {
		return switchResolved{kind: switchRev, name: input, input: input, commit: commit}, nil
	}
	hits := substringHits(input, heads, remotes)
	switch len(hits) {
	case 1:
		return switchResolved{hits[0].kind, hits[0].name, input, ""}, nil
	case 0:
		return switchResolved{}, notARef(input)
	default:
		names := make([]string, len(hits))
		for i, hit := range hits {
			names[i] = hit.name
		}
		return switchResolved{}, branchListError(fmt.Sprintf("'%s' matches more than one branch:", input), names)
	}
}

func resolveQualified(input string, kind switchKind, name string, heads, remotes, tags []string) (switchResolved, error) {
	if name == "" {
		return switchResolved{}, notARef(input)
	}
	switch kind {
	case switchLocal:
		if containsName(heads, name) {
			return switchResolved{switchLocal, name, input, ""}, nil
		}
	case switchRemote:
		if !isRemoteHEAD(name) && containsName(remotes, name) {
			return switchResolved{switchRemote, name, input, ""}, nil
		}
	case switchTag:
		if containsName(tags, name) {
			return resolveTag(name, input)
		}
	}
	return switchResolved{}, notARef(input)
}

func qualifiedKind(input string) (switchKind, string, bool) {
	prefixes := []struct {
		prefix string
		kind   switchKind
	}{
		{"refs/heads/", switchLocal},
		{"refs/remotes/", switchRemote},
		{"refs/tags/", switchTag},
		{"heads/", switchLocal},
		{"remotes/", switchRemote},
		{"tags/", switchTag},
	}
	for _, p := range prefixes {
		if !strings.HasPrefix(input, p.prefix) {
			continue
		}
		return p.kind, strings.TrimPrefix(input, p.prefix), true
	}
	return 0, "", false
}

func isExactLocalInput(input string) bool {
	heads := refList("refs/heads")
	if kind, name, ok := qualifiedKind(input); ok {
		return kind == switchLocal && containsName(heads, name)
	}
	return containsName(heads, input)
}

func noteOtherKinds(input string, winner switchKind, remotes, tags map[string]bool) {
	var forms []string
	if winner != switchRemote && remotes[input] {
		forms = append(forms, "remotes/"+input)
	}
	if winner != switchTag && tags[input] {
		forms = append(forms, "tags/"+input)
	}
	if len(forms) == 0 {
		return
	}
	text.InfoText("Also available as " + joinQuoted(forms) + ".")
}

func joinQuoted(forms []string) string {
	quoted := make([]string, len(forms))
	for i, form := range forms {
		quoted[i] = "'" + form + "'"
	}
	switch len(quoted) {
	case 1:
		return quoted[0]
	case 2:
		return quoted[0] + " and " + quoted[1]
	default:
		return strings.Join(quoted[:len(quoted)-1], ", ") + ", and " + quoted[len(quoted)-1]
	}
}

func remoteBranchesNamed(branch string, remotes []string) []string {
	var matches []string
	for _, short := range remotes {
		_, name, ok := strings.Cut(short, "/")
		if ok && name == branch {
			matches = append(matches, short)
		}
	}
	return matches
}

func substringHits(pattern string, heads, remotes []string) []switchHit {
	var hits []switchHit
	for _, name := range heads {
		if strings.Contains(name, pattern) {
			hits = append(hits, switchHit{switchLocal, name})
		}
	}
	for _, name := range remotes {
		if strings.Contains(name, pattern) {
			hits = append(hits, switchHit{switchRemote, name})
		}
	}
	return hits
}

func refList(prefix string) []string {
	raw := git.OutputOK("for-each-ref", "--format=%(refname)", prefix)
	base := prefix + "/"
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, base) {
			continue
		}
		name := strings.TrimPrefix(line, base)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

func remoteList() []string {
	all := refList("refs/remotes")
	out := make([]string, 0, len(all))
	for _, name := range all {
		if isRemoteHEAD(name) {
			continue
		}
		out = append(out, name)
	}
	return out
}

func refSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}
	return set
}

func containsName(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}

func isRemoteHEAD(short string) bool {
	_, branch, ok := strings.Cut(short, "/")
	return ok && branch == "HEAD"
}

func notARef(ref string) error {
	return workflowText(fmt.Sprintf("'%s' is not a branch, tag, or commit", ref))
}

func branchListError(header string, names []string) error {
	text.ErrorText(header)
	for _, name := range names {
		text.ErrorText(" - " + name)
	}
	return cliruntime.ExitStatus{Code: exitcode.WorkflowError}
}

func workflowText(msg string) error {
	text.ErrorText(msg)
	return cliruntime.ExitStatus{Code: exitcode.WorkflowError}
}
