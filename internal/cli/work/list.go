package work

import (
	"fmt"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var listID = cmdid.ID{Command: "work", Action: "list"}

func newListCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "Prints HEAD state",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cliruntime.RunWithWorkflows(cmd, listID, listRun)
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func listRun() error {
	branch := cliruntime.CurrentBranch()
	text.InfoText(">>> Branch refs:")
	text.InfoText("local:  " + branch)
	if upstream := state.UpstreamOf(branch); upstream != "" {
		text.InfoText("remote: " + upstream)
	} else {
		text.InfoText("remote: none")
	}
	text.InfoText("")
	latest := config.FreshestBranchSourceBranch(branch)
	if git.OutputOK("rev-list", latest+".."+branch) != "" {
		text.InfoText(fmt.Sprintf(">>> New commits (comparing to '%s' branch):", latest))
		if err := cliruntime.GitStdout("log", "--oneline", latest+".."+branch); err != nil {
			return err
		}
		text.InfoText("")
	}
	if status := git.OutputOK("status", "--short"); status != "" {
		text.InfoText(">>> Uncommitted modifications:")
		if err := cliruntime.GitStdout("status", "--short"); err != nil {
			return err
		}
		text.InfoText("")
	}
	if stashes := git.OutputOK("stash", "list"); stashes != "" {
		text.InfoText(">>> Available stashes:")
		if err := cliruntime.GitStdout("stash", "list"); err != nil {
			return err
		}
		text.InfoText("")
	}
	return nil
}
