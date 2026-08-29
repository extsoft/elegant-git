package work

import (
	"fmt"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

// NewCommand returns the work object command group.
func NewCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "work",
		Short: "Day-to-day contributions",
		RunE:  runBare,
	}
	c.AddCommand(newStartCommand())
	c.AddCommand(newSaveCommand())
	c.AddCommand(newAmendCommand())
	c.AddCommand(newListCommand())
	c.AddCommand(newPolishCommand())
	c.AddCommand(newSyncCommand())
	c.AddCommand(newPushCommand())
	c.AddCommand(newTrackCommand())
	c.AddCommand(newAcceptCommand())
	return c
}

func runBare(cmd *cobra.Command, _ []string) error {
	p := prompt.FromContext(cmd.Context())
	if prompt.NonInteractive(p) {
		return cliruntime.NewUsageError(cmd, fmt.Errorf("action is required"))
	}
	return runSession(cmd, inspect)
}
