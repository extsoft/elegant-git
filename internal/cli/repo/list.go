package repo

import (
	memorycmd "github.com/extsoft/elegant-git/internal/cli/memory"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show repository memory and registry state for the current repository",
		Long:  "Prints per-repository memory, registry linkage, and workspace association for the current repository.",
		RunE:  runList,
	}
}

func runList(cmd *cobra.Command, _ []string) error {
	return memorycmd.PrintRepoList(cmd.OutOrStdout())
}
