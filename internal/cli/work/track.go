package work

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var trackID = cmdid.ID{Command: "work", Action: "track"}

func newTrackCommand() *cobra.Command {
	spec := trackSpec()
	c := &cobra.Command{
		Use:   "track <name> [local-branch]",
		Short: "Checks out a remote-tracking branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, trackID, func() error {
				return trackRun(cmd, args)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func trackSpec() argspec.Spec {
	var pattern string
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInputWithComplete("name", 0, true, "Remote branch name or pattern", &pattern, nil, sources.RemoteBranches, false),
	}}
}

func trackRun(cmd *cobra.Command, args []string) error {
	pattern := ""
	if len(args) >= 1 {
		pattern = args[0]
	}
	localBranch := ""
	if len(args) >= 2 {
		localBranch = args[1]
	}
	p := prompt.FromContext(cmd.Context())
	if pattern == "" && prompt.NonInteractive(p) {
		return &argspec.ErrMissingRequired{Names: []string{"name"}}
	}
	return trackLogic(cmd.Context(), p, pattern, localBranch)
}

func trackExactRemote(remoteRef, localBranch string) error {
	if err := git.Verbose("fetch", "--all"); err != nil {
		return err
	}
	remotes := strings.Split(git.OutputOK("for-each-ref", "--format=%(refname:short)", "refs/remotes"), "\n")
	found := false
	for _, ref := range remotes {
		if strings.TrimSpace(ref) == remoteRef {
			found = true
			break
		}
	}
	if !found {
		cliruntime.ExitWorkflowError(fmt.Sprintf("There is no remote branch %q.", remoteRef))
	}
	local := localBranch
	if local == "" {
		local = cliruntime.BranchFromRemoteBranch(remoteRef)
	}
	return git.Verbose("checkout", "-B", local, remoteRef)
}

func trackLogic(ctx context.Context, p prompt.Prompter, pattern, localBranch string) error {
	if err := git.Verbose("fetch", "--all"); err != nil {
		return err
	}

	pattern, err := resolveRemotePattern(ctx, p, pattern)
	if err != nil {
		return err
	}

	matches := remoteBranchesMatching(pattern)
	if len(matches) > 1 {
		text.InfoText("The following branches are found:")
		for _, b := range matches {
			text.InfoText(" - " + b)
		}
		cliruntime.ExitWorkflowError("Please re-run the command with concrete branch name from the list above!")
	}
	if len(matches) == 0 {
		cliruntime.ExitWorkflowError(fmt.Sprintf("There is no branch that matches the '%s' pattern.", pattern))
	}
	remote := matches[0]

	local, err := resolveLocalBranch(p, remote, localBranch)
	if err != nil {
		return err
	}
	return git.Verbose("checkout", "-B", local, remote)
}

func resolveRemotePattern(ctx context.Context, p prompt.Prompter, pattern string) (string, error) {
	if strings.TrimSpace(pattern) != "" {
		return strings.TrimSpace(pattern), nil
	}
	choices, err := sources.RemoteBranches(ctx)
	if err != nil {
		return "", err
	}
	if len(choices) == 0 {
		return "", &argspec.ErrEmptySource{Name: "name"}
	}
	promptChoices := make([]prompt.Choice, len(choices))
	for i, c := range choices {
		promptChoices[i] = prompt.Choice{Value: c.Value, Description: c.Description}
	}
	val, err := p.Pick("Remote branch name or pattern", promptChoices, "")
	if err != nil {
		return "", err
	}
	val = strings.TrimSpace(val)
	if i := strings.IndexByte(val, '\t'); i >= 0 {
		val = val[:i]
	}
	if val == "" {
		return "", errors.New("name is required")
	}
	return val, nil
}

func resolveLocalBranch(p prompt.Prompter, remote, localBranch string) (string, error) {
	suggested := cliruntime.BranchFromRemoteBranch(remote)
	if prompt.NonInteractive(p) {
		if strings.TrimSpace(localBranch) != "" {
			return strings.TrimSpace(localBranch), nil
		}
		return suggested, nil
	}
	if strings.TrimSpace(localBranch) != "" {
		suggested = strings.TrimSpace(localBranch)
	}
	val, err := p.EditOrAccept("Local branch name", suggested)
	if err != nil {
		return "", err
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return suggested, nil
	}
	return val, nil
}

func remoteBranchesMatching(pattern string) []string {
	remotes := strings.Split(git.OutputOK("for-each-ref", "--format=%(refname:short)", "refs/remotes"), "\n")
	var matches []string
	for _, ref := range remotes {
		ref = strings.TrimSpace(ref)
		if ref != "" && strings.Contains(ref, pattern) {
			matches = append(matches, ref)
		}
	}
	return matches
}
