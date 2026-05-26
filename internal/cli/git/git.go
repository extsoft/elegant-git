package git

import (
	"github.com/spf13/cobra"
)

// NewCommand returns the git object command group.
func NewCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "git",
		Short: "Configure Git installation",
	}
	c.AddCommand(newConfigureCommand())
	c.AddCommand(newMigrateCommand())
	c.AddCommand(newStatusCommand())
	return c
}
