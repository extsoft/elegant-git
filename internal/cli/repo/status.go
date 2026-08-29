package repo

import (
	memorycmd "github.com/extsoft/elegant-git/internal/cli/memory"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show repository memory and registry state for the current repository",
		Long:  "Prints per-repository memory, registry linkage, and workspace association for the current repository.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return memorycmd.PrintRepoStatus(cmd.OutOrStdout())
		},
	}
}
