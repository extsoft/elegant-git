package version

import (
	"fmt"

	"github.com/extsoft/elegant-git/internal/version"
	"github.com/spf13/cobra"
)

// NewCommand returns the version subcommand.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the program version",
		Long:  "Prints the installed git-elegant version string.",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(version.Version)
		},
	}
}
