package cli

import (
	"fmt"
	"os"

	"github.com/bees-hive/elegant-git/internal/workflows"
	"github.com/spf13/cobra"
)

func stubRunE(cmd *cobra.Command, args []string) error {
	workflows.RunAhead(cmd.Name())
	defer workflows.RunAfter(cmd.Name())
	fmt.Fprintf(os.Stderr, "git elegant %s: not implemented yet\n", cmd.Name())
	return fmt.Errorf("not implemented")
}

func newStubCommand(spec commandSpec) *cobra.Command {
	description := spec.description
	if description == "" {
		description = spec.purpose
	}
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  description,
		RunE:  stubRunE,
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}
