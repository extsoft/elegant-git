package cli

import (
	"fmt"
	"strings"

	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/pipe"
	"github.com/bees-hive/elegant-git/internal/state"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newPolishWorkCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWithWorkflows(cmd, polishWorkRun)
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func polishWorkRun() error {
	branch := currentBranch()
	if config.IsBranchProtected(branch) {
		exitProtectedNoRewrite(branch)
	}
	if state.IsThereActiveRebase() {
		return git.Verbose("rebase", "--continue")
	}
	latest := config.FreshestDefaultBranch()
	commits := strings.Fields(git.OutputOK("rev-list", latest+"..@"))
	if len(commits) == 0 {
		text.InfoText(fmt.Sprintf("There are no new commits comparing to '%s' branch.", latest))
		return nil
	}
	n := len(commits)
	return pipe.StashPipe("polish-work", func() error {
		return git.Verbose("rebase", "--interactive", fmt.Sprintf("@~%d", n))
	})
}
