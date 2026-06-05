package work

import (
	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/cli/completion"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/pipe"
	"github.com/bees-hive/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

var pushID = cmdid.ID{Command: "work", Action: "push"}

func pushSpec(branch *string) argspec.Spec {
	in := argspec.PositionalInput("branch-name", 0, false, "Remote branch name", branch, nil)
	in.OmitInteractive = true
	return argspec.Spec{Inputs: []argspec.Input{in}}
}

func newPushCommand() *cobra.Command {
	var branch string
	spec := pushSpec(&branch)
	c := &cobra.Command{
		Use:   "push [branch-name]",
		Short: "Publishes HEAD to a remote repository",
		Long:  "Rebases onto upstream when needed, then force-pushes the current branch to its remote tracking branch or the given branch name.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, pushID, func() error {
				return pushRun(cmd, args)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.AttachArgs(c, spec)
	return c
}

func pushRun(cmd *cobra.Command, args []string) error {
	var remoteBranchArg string
	if err := argspec.ResolveCmd(cmd, args, pushSpec(&remoteBranchArg)); err != nil {
		return err
	}
	branch := cliruntime.CurrentBranch()
	if config.IsBranchProtected(branch) {
		cliruntime.ExitProtectedDeliver(branch)
	}
	return pipe.StashPipe(pushID, func() error {
		return pushLogic(branch, remoteBranchArg)
	})
}

func pushLogic(localBranch, remoteBranchArg string) error {
	if state.IsThereActiveRebase() {
		if err := git.Verbose("rebase", "--continue"); err != nil {
			return err
		}
	} else {
		if err := git.Verbose("fetch"); err != nil {
			return err
		}
		if err := git.Verbose("rebase", config.DefaultRemoteTrackingBranch()); err != nil {
			return err
		}
	}
	upstream := cliruntime.BranchUpstreamShort(localBranch)
	branch := cliruntime.BranchFromRemoteBranch(upstream)
	remote := cliruntime.RemoteFromRemoteBranch(upstream)
	if remoteBranchArg != "" {
		branch = remoteBranchArg
	}
	if branch == "" {
		branch = localBranch
	}
	if remote == "" {
		remote = config.DefaultUpstreamRemote
	}
	refSpec := localBranch + ":" + branch
	return git.VerboseOpLines(cliruntime.OpenURLsIfPossible, "push", "--set-upstream", "--force", remote, refSpec)
}
