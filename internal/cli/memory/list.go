package memory

import (
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "Summarize elegant-git memory stores",
		Long:  "Prints shared memory paths, workspace and repository counts, and hints for detail commands.",
		RunE:  runList,
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func runList(cmd *cobra.Command, _ []string) error {
	return PrintMemorySummary(cmd.OutOrStdout())
}
