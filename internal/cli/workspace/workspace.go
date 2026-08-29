package workspace

import (
	"fmt"

	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

// NewCommand returns the workspace command group.
func NewCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "workspace",
		Short: "Manage git workspaces",
		RunE:  runBare,
	}
	c.AddCommand(newListCommand())
	c.AddCommand(newNewCommand())
	c.AddCommand(newLegacyCreateCommand())
	c.AddCommand(newLinkCommand())
	c.AddCommand(newEditCommand())
	c.AddCommand(newDeleteCommand())
	c.AddCommand(newStatusCommand())
	c.AddCommand(newFetchCommand())
	c.AddCommand(newDoctorCommand())
	return c
}

func runBare(cmd *cobra.Command, _ []string) error {
	p := prompt.FromContext(cmd.Context())
	if prompt.NonInteractive(p) {
		return cliruntime.NewUsageError(cmd, fmt.Errorf("action is required"))
	}
	return runSession(cmd, inspect)
}
