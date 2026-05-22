package cli

import (
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/pipe"
	"github.com/bees-hive/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

func newDeliverWorkCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithWorkflows(cmd, func() error {
				return deliverWorkRun(args)
			})
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func deliverWorkRun(args []string) error {
	branch := currentBranch()
	if config.IsBranchProtected(branch) {
		exitProtectedDeliver(branch)
	}
	remoteBranchArg := ""
	if len(args) > 0 {
		remoteBranchArg = args[0]
	}
	return pipe.StashPipe("deliver-work", func() error {
		return deliverWorkLogic(branch, remoteBranchArg)
	})
}

func deliverWorkLogic(localBranch, remoteBranchArg string) error {
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
	upstream := branchUpstreamShort(localBranch)
	branch := branchFromRemoteBranch(upstream)
	remote := remoteFromRemoteBranch(upstream)
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
	return git.VerboseOp(openURLsIfPossible, "push", "--set-upstream", "--force", remote, refSpec)
}
