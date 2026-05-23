package hook

import "github.com/spf13/cobra"

// NewCommand returns the hook object command group.
func NewCommand() *cobra.Command {
	c := &cobra.Command{Use: "hook", Short: "Manage command hooks"}
	c.AddCommand(newListCommand())
	c.AddCommand(newNewCommand())
	c.AddCommand(newEditCommand())
	c.AddCommand(newMigrateCommand())
	return c
}
