package memory

import (
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "status",
		Short: "Summarize elegant-git memory stores",
		Long:  "Prints shared memory paths, profile and repository counts, and hints for detail commands.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return PrintMemorySummary(cmd.OutOrStdout())
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}
