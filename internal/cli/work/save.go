package work

import (
	"fmt"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var saveID = cmdid.ID{Command: "work", Action: "save"}

func newSaveCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "save",
		Short: "Commits current modifications",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cliruntime.RunWithWorkflows(cmd, saveID, func() error {
				return saveRun(cmd)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func saveRun(cmd *cobra.Command) error {
	branch := cliruntime.CurrentBranch()
	if config.IsBranchProtected(branch) {
		if cliruntime.StdinIsInteractive(cmd.Context()) {
			return saveViaStartOnProtected(cmd, branch)
		}
		cliruntime.ExitProtectedNoCommits(branch)
	}
	return saveCommit()
}

func saveViaStartOnProtected(cmd *cobra.Command, branch string) error {
	text.InfoBox(fmt.Sprintf("Warning: no direct commits on the protected '%s' branch.", branch))
	text.InfoText("Starting `eg work start` to create a feature branch first.")
	var name, fromRef string
	spec := startSpec(&name, &fromRef)
	if err := cliruntime.RunWithWorkflows(cmd, startID, func() error {
		return startRun(cmd, nil, spec)
	}); err != nil {
		return err
	}
	return saveCommit()
}

func saveCommit() error {
	if err := git.Verbose("add", "--interactive"); err != nil {
		return err
	}
	return git.Verbose("commit")
}
