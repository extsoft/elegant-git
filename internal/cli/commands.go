package cli

import (
	clicatalog "github.com/extsoft/elegant-git/internal/cli/catalog"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/spf13/cobra"
)

// AllCanonicalCommandIDs returns object.action ids for every registered subcommand.
func AllCanonicalCommandIDs() []string {
	return clicatalog.AllCanonicalCommandIDs()
}

// AttachObjectHelp sets HelpFunc so `eg <object> --help` lists actions.
func AttachObjectHelp(c *cobra.Command, object string) {
	clicatalog.AttachObjectHelp(c, object)
}

// AttachObjectGroup sets Run and HelpFunc so `eg <object>` shows that object's subcommands.
func AttachObjectGroup(c *cobra.Command, object string) {
	clicatalog.AttachObjectGroup(c, object)
}

func newBaseCommand(use, short, long string) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		Long:          long,
		SilenceUsage:  false,
		SilenceErrors: true,
	}
}

func attachHelp(c *cobra.Command) {
	c.SetHelpFunc(cliruntime.CommandHelp)
}
