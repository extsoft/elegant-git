package work

import (
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/spf13/cobra"
)

var amendID = cmdid.ID{Command: "work", Action: "amend"}

func newAmendCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "amend",
		Short: "Amends the last commit",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cliruntime.RunWithWorkflows(cmd, amendID, amendRun)
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func amendRun() error {
	branch := cliruntime.CurrentBranch()
	if config.IsBranchProtected(branch) {
		cliruntime.ExitProtectedNoCommits(branch)
	}
	if err := git.Verbose("add", "--interactive"); err != nil {
		return err
	}
	return git.Verbose("commit", "--amend")
}
