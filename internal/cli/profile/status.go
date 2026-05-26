package profile

import (
	memorycmd "github.com/bees-hive/elegant-git/internal/cli/memory"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the linked profile for the current repository",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return memorycmd.PrintProfileStatus(cmd.OutOrStdout())
		},
	}
}
