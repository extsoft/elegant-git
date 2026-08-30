package git

import (
	memorycmd "github.com/extsoft/elegant-git/internal/cli/memory"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show global Git installation and shared memory state",
		Long:  "Prints global Git configuration, elegant-git memory paths, and workspace/repository counts.",
		RunE:  runList,
	}
}

func runList(cmd *cobra.Command, _ []string) error {
	return memorycmd.PrintGitList(cmd.OutOrStdout())
}
