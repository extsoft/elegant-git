package cli

import (
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/spf13/cobra"
)

func newAmendWorkCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWithWorkflows(cmd, amendWorkRun)
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func amendWorkRun() error {
	branch := currentBranch()
	if config.IsBranchProtected(branch) {
		exitProtectedNoCommits(branch)
	}
	if err := git.Verbose("add", "--interactive"); err != nil {
		return err
	}
	if err := git.Verbose("diff", "--cached", "--check"); err != nil {
		return err
	}
	return git.Verbose("commit", "--amend")
}
