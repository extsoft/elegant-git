package cli

import (
	"fmt"
	"os"

	"github.com/bees-hive/elegant-git/internal/workflows"
	"github.com/spf13/cobra"
)

func newShowWorkflowsCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWithWorkflows(cmd, showWorkflowsRun)
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func showWorkflowsRun() error {
	for _, command := range CommandNames() {
		listWorkflowIfExists(workflows.PersonalWorkflowsFile(command, "ahead"))
		listWorkflowIfExists(workflows.PersonalWorkflowsFile(command, "after"))
		listWorkflowIfExists(workflows.CommonWorkflowsFile(command, "ahead"))
		listWorkflowIfExists(workflows.CommonWorkflowsFile(command, "after"))
	}
	return nil
}

func listWorkflowIfExists(path string) {
	if path == "" {
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return
	}
	fmt.Fprintln(os.Stdout, path)
}
