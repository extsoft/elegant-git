package work

import (
	"fmt"
	"io"
	"strings"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/statefmt"
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
			return cliruntime.RunWithWorkflows(cmd, listID, func() error {
				return listRun(cmd.OutOrStdout())
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func listRun(w io.Writer) error {
	branch := cliruntime.CurrentBranch()
	remote := "none"
	if upstream := state.UpstreamOf(branch); upstream != "" {
		remote = upstream
	}
	text.Finfo(w, "Branch")
	statefmt.PrintFields(w, "", []statefmt.Field{
		{Key: "local", Value: branch},
		{Key: "remote", Value: remote},
	})

	var steps []statefmt.Step
	status, _ := gitColoredOutput(w, "status", "--short")
	if strings.TrimSpace(status) != "" {
		fmt.Fprintln(w)
		text.Finfo(w, "Modifications")
		statefmt.PrintLines(w, status)
		steps = append(steps, statefmt.Step{Command: "eg work save", Comment: "commit the uncommitted modifications"})
	}
	latest := config.FreshestBranchSourceBranch(branch)
	if git.OutputOK("rev-list", latest+".."+branch) != "" {
		fmt.Fprintln(w)
		text.Finfo(w, fmt.Sprintf("Commits (comparing to %s)", latest))
		logOut, _ := gitColoredOutput(w, "log", "--oneline", latest+".."+branch)
		statefmt.PrintLines(w, logOut)
		steps = append(steps, statefmt.Step{Command: "eg work polish", Comment: "rewrite the unique commits"})
	}
	stashes, _ := gitColoredOutput(w, "stash", "list")
	if strings.TrimSpace(stashes) != "" {
		fmt.Fprintln(w)
		text.Finfo(w, "Stashes")
		statefmt.PrintLines(w, stashes)
	}
	statefmt.PrintFurtherSteps(w, steps)
	return nil
}

func gitColoredOutput(w io.Writer, args ...string) (string, error) {
	return git.Output(gitColorArgs(text.Colored(w), args...)...)
}

func gitColorArgs(color bool, args ...string) []string {
	if !color {
		return args
	}
	out := make([]string, 0, len(args)+2)
	out = append(out, "-c", "color.ui=always")
	return append(out, args...)
}
