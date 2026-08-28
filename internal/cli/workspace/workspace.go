package workspace

import "github.com/spf13/cobra"

// NewCommand returns the workspace command group.
func NewCommand() *cobra.Command {
	c := &cobra.Command{Use: "workspace", Short: "Manage git workspaces"}
	c.AddCommand(newStatusCommand())
	c.AddCommand(newCreateCommand())
	c.AddCommand(newEditCommand())
	c.AddCommand(newDeleteCommand())
	return c
}
