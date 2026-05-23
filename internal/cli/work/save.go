package work

import (
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/spf13/cobra"
)

var saveID = cmdid.ID{Command: "work", Action: "save"}

func newSaveCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "save",
		Short: "Commits current modifications",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cliruntime.RunWithWorkflows(cmd, saveID, saveRun)
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func saveRun() error {
	branch := cliruntime.CurrentBranch()
	if config.IsBranchProtected(branch) {
		cliruntime.ExitProtectedNoCommits(branch)
	}
	if err := git.Verbose("add", "--interactive"); err != nil {
		return err
	}
	if err := git.Verbose("diff", "--cached", "--check"); err != nil {
		return err
	}
	return git.Verbose("commit")
}
