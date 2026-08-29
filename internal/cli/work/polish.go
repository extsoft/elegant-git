package work

import (
	"fmt"
	"strings"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/pipe"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var polishID = cmdid.ID{Command: "work", Action: "polish"}

func newPolishCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "polish",
		Short: "Rebases HEAD interactively",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cliruntime.RunWithWorkflows(cmd, polishID, polishRun)
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func polishRun() error {
	branch := cliruntime.CurrentBranch()
	if config.IsBranchProtected(branch) {
		cliruntime.ExitProtectedNoRewrite(branch)
	}
	if state.IsThereActiveRebase() {
		return git.Verbose("rebase", "--continue")
	}
	latest := config.FreshestBranchSourceBranch(branch)
	commits := strings.Fields(git.OutputOK("rev-list", latest+"..@"))
	if len(commits) == 0 {
		text.InfoText(fmt.Sprintf("There are no new commits comparing to '%s' branch.", latest))
		return nil
	}
	n := len(commits)
	return pipe.StashPipe(polishID, func() error {
		return git.Verbose("rebase", "--interactive", fmt.Sprintf("@~%d", n))
	})
}
