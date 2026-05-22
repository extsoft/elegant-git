package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newPolishWorkflowCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithWorkflows(cmd, func() error {
				return polishWorkflowRun(args)
			})
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func polishWorkflowRun(args []string) error {
	requireArgs(args, "Please specify a workflow file name")
	path := args[0]
	if _, err := os.Stat(path); err != nil {
		exitWorkflowError(fmt.Sprintf("The '%s' file does not exist.", path))
	}
	return openInEditor(path)
}
