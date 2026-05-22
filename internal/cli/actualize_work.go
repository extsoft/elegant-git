package cli

import (
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/pipe"
	"github.com/bees-hive/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

func newActualizeWorkCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithWorkflows(cmd, func() error {
				return actualizeWorkRun(args)
			})
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func actualizeWorkRun(args []string) error {
	return pipe.StashPipe("actualize-work", func() error {
		return actualizeWorkLogic(args)
	})
}

func actualizeWorkLogic(args []string) error {
	if state.IsThereActiveRebase() {
		if err := git.Verbose("rebase", "--continue"); err != nil {
			return err
		}
	}
	branchArg := ""
	if len(args) > 0 && args[0] != "" {
		branchArg = args[0]
	}
	if branchArg != "" {
		if state.IsRemoteBranch(branchArg) {
			fetchOrInform()
			return git.Verbose("rebase", branchArg)
		}
		return git.Verbose("rebase", branchArg)
	}
	if state.AreThereRemotes() {
		fetchOrInform()
		return git.Verbose("rebase", config.DefaultRemoteTrackingBranch())
	}
	return git.Verbose("rebase", config.DefaultBranch())
}
