package work

import (
	"context"
	"fmt"
	"strings"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/exitcode"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/pipe"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var switchID = cmdid.ID{Command: "work", Action: "switch"}

var switchChangeChoices = []prompt.Choice{
	{Value: "add", Description: "Keep changes and move them onto the new branch."},
	{Value: "reset", Description: "Discard uncommitted changes."},
	{Value: "cancel", Description: "Cancel work switch."},
}

func newSwitchCommand() *cobra.Command {
	spec := switchSpec()
	c := &cobra.Command{
		Use:   "switch [ref] [local-branch]",
		Short: "Switches to a branch, or creates one from a remote branch, tag, or revision.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, switchID, func() error {
				return switchRun(cmd, args)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func switchSpec() argspec.Spec {
	var ref, local string
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInputWithComplete("ref", 0, true, "Branch, remote branch, or tag", &ref, nil, sources.RefsListed, false),
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
	if strings.HasPrefix(ref, "-") {
		return cliruntime.NewUsageError(cmd, fmt.Errorf("ref must not start with '-'"))
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
	}
	return switchTo(cmd, p, ref, localBranch, picked)
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
	if len(choices) == 0 {
		return "", &argspec.ErrEmptySource{Name: "ref"}
	}
	promptChoices := make([]prompt.Choice, len(choices))
	for i, c := range choices {
		promptChoices[i] = prompt.Choice{Value: c.Value, Description: c.Description}
	}
	val, err := p.Pick("Branch, remote branch, or tag", promptChoices, "")
	if err != nil {
		return "", err
	}
	val = strings.TrimSpace(val)
	if i := strings.IndexByte(val, '\t'); i >= 0 {
		val = val[:i]
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
			return cliruntime.NewUsageError(cmd, fmt.Errorf("local-branch applies only to a remote branch"))
		}
		if cliruntime.CurrentBranch() == resolved.name {
			text.InfoText(fmt.Sprintf("Already on '%s'", resolved.name))
			return nil
		}
	}
	return withSwitchChanges(cmd.Context(), func() error {
		return switchAct(p, resolved, localBranch)
	})
}

func switchAct(p prompt.Prompter, resolved switchResolved, localBranch string) error {
	switch resolved.kind {
	case switchLocal:
		return git.Verbose("checkout", resolved.name, "--")
	case switchRemote:
		local, fold, err := resolveLocalBranch(p, resolved.name, localBranch)
		if err != nil {
			return err
		}
		return checkoutTrackedLocal(p, local, resolved.name, fold, trackDivergenceStay)
	default:
		return notABranch(resolved.input)
	}
}

func withSwitchChanges(ctx context.Context, logic func() error) error {
	if state.HeadUnborn() {
		if strings.TrimSpace(git.OutputOK("status", "--porcelain")) != "" {
			return workflowText("Uncommitted changes cannot be stashed because this repository has no commits yet.")
		}
		return logic()
	}
	mode, err := resolveWorkChanges(ctx, switchChangeChoices)
	if err != nil {
		return err
	}
	switch mode {
	case "none":
		return logic()
	case "stash":
		return pipe.StashPipeRestoring(switchID, logic)
	case "reset":
		if err := git.Verbose("reset", "--hard", "HEAD"); err != nil {
			return err
		}
		return logic()
	case "cancel":
		return fmt.Errorf("work switch cancelled")
	default:
		return fmt.Errorf("invalid choice %q (use add, reset, or cancel)", mode)
	}
}

type switchKind int

const (
	switchLocal switchKind = iota
	switchRemote
	switchTag
)

type switchResolved struct {
	kind  switchKind
	name  string
	input string
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
		return resolveQualified(input, kind, name, heads, remotes)
	}
	headSet := refSet(heads)
	remoteSet := refSet(remotes)
	tagSet := refSet(tags)
	if headSet[input] {
		noteOtherKinds(input, switchLocal, remoteSet, tagSet)
		return switchResolved{switchLocal, input, input}, nil
	}
	if remoteSet[input] {
		noteOtherKinds(input, switchRemote, remoteSet, tagSet)
		return switchResolved{switchRemote, input, input}, nil
	}
	if matches := remoteBranchesNamed(input, remotes); len(matches) == 1 {
		noteOtherKinds(input, switchRemote, remoteSet, tagSet)
		return switchResolved{switchRemote, matches[0], input}, nil
	} else if len(matches) > 1 {
		return switchResolved{}, branchListError(fmt.Sprintf("'%s' matches more than one remote branch:", input), matches)
	}
	if tagSet[input] {
		return switchResolved{}, notABranch(input)
	}
	hits := substringHits(input, heads, remotes)
	switch len(hits) {
	case 1:
		return switchResolved{hits[0].kind, hits[0].name, input}, nil
	case 0:
		return switchResolved{}, notABranch(input)
	default:
		names := make([]string, len(hits))
		for i, hit := range hits {
			names[i] = hit.name
		}
		return switchResolved{}, branchListError(fmt.Sprintf("'%s' matches more than one branch:", input), names)
	}
}

func resolveQualified(input string, kind switchKind, name string, heads, remotes []string) (switchResolved, error) {
	if name == "" {
		return switchResolved{}, notABranch(input)
	}
	switch kind {
	case switchLocal:
		if containsName(heads, name) {
			return switchResolved{switchLocal, name, input}, nil
		}
	case switchRemote:
		if !isRemoteHEAD(name) && containsName(remotes, name) {
			return switchResolved{switchRemote, name, input}, nil
		}
	}
	return switchResolved{}, notABranch(input)
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

func notABranch(ref string) error {
	return workflowText(fmt.Sprintf("'%s' is not a branch", ref))
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
