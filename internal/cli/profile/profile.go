package profile

import "github.com/spf13/cobra"

// NewCommand returns the profile command group.
func NewCommand() *cobra.Command {
	c := &cobra.Command{Use: "profile", Short: "Manage git user profiles"}
	c.AddCommand(newStatusCommand())
	c.AddCommand(newCreateCommand())
	c.AddCommand(newEditCommand())
	c.AddCommand(newDeleteCommand())
	return c
}
