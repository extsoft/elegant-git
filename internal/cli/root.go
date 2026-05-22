package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/bees-hive/elegant-git/internal/exitcode"
	"github.com/bees-hive/elegant-git/internal/version"
	"github.com/bees-hive/elegant-git/internal/workflows"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "git-elegant",
	Short:         "An assistant who carefully automates routine work with Git.",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version.Version,
	Run: func(cmd *cobra.Command, _ []string) {
		writeRootUsage(cmd.OutOrStdout())
	},
}

// Execute runs the git-elegant CLI.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if isUnknownCommand(err) {
			name := unknownCommandName(err)
			fmt.Fprintf(os.Stderr, "Unknown command: git elegant %s\n", name)
			writeRootUsage(os.Stderr)
			os.Exit(exitcode.UnknownCommand)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeRootUsage(cmd.OutOrStdout())
	})
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	rootCmd.PersistentFlags().BoolVar(&workflows.Skip, "no-workflows", false, "disables available workflows")

	rootCmd.AddCommand(&cobra.Command{
		Use:    "version",
		Hidden: true,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(version.Version)
		},
	})

	for _, spec := range allCommandSpecs() {
		rootCmd.AddCommand(newStubCommand(spec))
	}
}

func isUnknownCommand(err error) bool {
	return strings.Contains(err.Error(), "unknown command")
}

func unknownCommandName(err error) string {
	msg := err.Error()
	const prefix = `unknown command "`
	if i := strings.Index(msg, prefix); i >= 0 {
		start := i + len(prefix)
		if j := strings.Index(msg[start:], `"`); j >= 0 {
			return msg[start : start+j]
		}
	}
	return ""
}
