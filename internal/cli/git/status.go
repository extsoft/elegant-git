package git

import (
	memorycmd "github.com/bees-hive/elegant-git/internal/cli/memory"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show global Git installation and shared memory state",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return memorycmd.PrintGitStatus(cmd.OutOrStdout())
		},
	}
}
