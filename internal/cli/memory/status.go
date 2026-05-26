package memory

import "github.com/spf13/cobra"

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Summarize elegant-git memory stores",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return PrintMemorySummary(cmd.OutOrStdout())
		},
	}
}
