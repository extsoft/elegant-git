package memory

import "github.com/spf13/cobra"

// NewCommand returns the memory command group.
func NewCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "memory",
		Short: "Inspect elegant-git memory stores",
	}
	c.AddCommand(newStatusCommand())
	c.AddCommand(newProfilesCommand())
	c.AddCommand(newRepositoriesCommand())
	return c
}
