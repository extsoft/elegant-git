package self

import "github.com/spf13/cobra"

// NewCommand returns the self object command group.
func NewCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "self",
		Short: "Set up and inspect Elegant Git",
	}
	c.AddCommand(newConfigureCommand())
	c.AddCommand(newListCommand())
	c.AddCommand(newDoctorCommand())
	return c
}
