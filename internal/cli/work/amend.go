package work

import (
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/spf13/cobra"
)

var amendID = cmdid.ID{Command: "work", Action: "amend"}

func newAmendCommand() *cobra.Command {
	c := &cobra.Command{
		Use:    "amend",
		Short:  "Amends the last commit",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cliruntime.RunWithWorkflows(cmd, amendID, amendRun)
		},
	}
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[deprecation.SurfaceAnnotation] = "work amend"
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
