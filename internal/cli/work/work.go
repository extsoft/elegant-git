package work

import "github.com/spf13/cobra"

// NewCommand returns the work object command group.
func NewCommand() *cobra.Command {
	c := &cobra.Command{Use: "work", Short: "Day-to-day contributions"}
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
