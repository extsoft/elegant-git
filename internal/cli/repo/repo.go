package repo

import (
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

// NewCommand returns the repo object command group.
func NewCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "repo",
		Short: "Manage repositories",
		RunE:  runBare,
	}
	c.AddCommand(newConfigureCommand())
	c.AddCommand(newCloneCommand())
	c.AddCommand(newInitCommand())
	c.AddCommand(newPruneCommand())
	c.AddCommand(newMigrateCommand())
	c.AddCommand(newListCommand())
	c.AddCommand(newStatusCommand())
	c.AddCommand(newSyncCommand())
	c.AddCommand(newDoctorCommand())
	return c
}

func runBare(cmd *cobra.Command, _ []string) error {
	p := prompt.FromContext(cmd.Context())
	if prompt.NonInteractive(p) {
		return cmd.Help()
	}
	return runSession(cmd, inspect)
}
