package repo

import "github.com/spf13/cobra"

// NewCommand returns the repo object command group.
func NewCommand() *cobra.Command {
	c := &cobra.Command{Use: "repo", Short: "Manage repositories"}
	c.AddCommand(newConfigureCommand())
	c.AddCommand(newCloneCommand())
	c.AddCommand(newInitCommand())
	c.AddCommand(newPruneCommand())
	c.AddCommand(newMigrateCommand())
	c.AddCommand(newStatusCommand())
	c.AddCommand(newSyncCommand())
	c.AddCommand(newRelocateCommand())
	return c
}
