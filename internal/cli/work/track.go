package work

import (
	"fmt"
	"strings"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var trackID = cmdid.ID{Command: "work", Action: "track"}

func newTrackCommand() *cobra.Command {
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
	return c
}

func trackRun(cmd *cobra.Command, args []string) error {
	var pattern, localBranch string
	if err := argspec.ResolveCmd(cmd, args, argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInput("name", 0, true, "Remote branch name or pattern", &pattern, nil),
		argspec.PositionalInput("local-branch", 1, false, "Local branch name", &localBranch, nil),
	}}); err != nil {
		return err
	}
	return trackLogic(pattern, localBranch)
}

func trackLogic(pattern, localBranch string) error {
	if err := git.Verbose("fetch", "--all"); err != nil {
		return err
	}
	remotes := strings.Split(git.OutputOK("for-each-ref", "--format=%(refname:short)", "refs/remotes"), "\n")
	var matches []string
	for _, ref := range remotes {
		ref = strings.TrimSpace(ref)
		if ref != "" && strings.Contains(ref, pattern) {
			matches = append(matches, ref)
		}
	}
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
	local := localBranch
	if local == "" {
		local = cliruntime.BranchFromRemoteBranch(remote)
	}
	return git.Verbose("checkout", "-B", local, remote)
}
