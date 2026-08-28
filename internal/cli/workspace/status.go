package workspace

import (
	memorycmd "github.com/bees-hive/elegant-git/internal/cli/memory"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the linked workspace for the current repository",
		Long:  "Prints the workspace linked to the current repository from shared memory.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return memorycmd.PrintWorkspaceStatus(cmd.OutOrStdout())
		},
	}
}
