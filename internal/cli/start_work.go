package cli

import (
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/pipe"
	"github.com/bees-hive/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

func newStartWorkCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithWorkflows(cmd, func() error {
				return startWorkRun(args)
			})
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func startWorkRun(args []string) error {
	requireArgs(args, "Please give a name for the new branch.")
	name := args[0]
	target := config.DefaultBranch()
	if len(args) > 1 && args[1] != "" {
		target = args[1]
	}
	return pipe.StashPipe("start-work", func() error {
		if err := git.Verbose("checkout", target); err != nil {
			return err
		}
		if state.IsThereUpstreamFor(target) {
			pullOrInform()
		}
		return git.Verbose("checkout", "-b", name)
	})
}
