package release

import "github.com/spf13/cobra"

// NewCommand returns the release object command group.
func NewCommand() *cobra.Command {
	c := &cobra.Command{Use: "release", Short: "Manage releases"}
	c.AddCommand(newNewCommand())
	c.AddCommand(newNotesCommand())
	return c
}
