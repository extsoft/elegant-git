package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newShowCommandsCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWithWorkflows(cmd, showCommandsRun)
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func showCommandsRun() error {
	for _, name := range CommandNames() {
		fmt.Fprintln(os.Stdout, name)
	}
	return nil
}
