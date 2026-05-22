package cli

import (
	"bufio"
	"os"

	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/spf13/cobra"
)

func newAcquireRepositoryCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWithWorkflows(cmd, acquireRepositoryRun)
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func acquireRepositoryRun() error {
	reader := bufio.NewReader(os.Stdin)
	if err := config.ObsoleteConfigurationsRemoving("--local"); err != nil {
		return err
	}
	if err := config.RepositoryBasicsConfiguration("--local", reader); err != nil {
		return err
	}
	if !config.IsGitAcquired() {
		if err := config.StandardsConfiguration("--local"); err != nil {
			return err
		}
		if err := config.AliasesConfiguration("--local", CommandNames()...); err != nil {
			return err
		}
	}
	return config.ConfigureSignature(reader)
}
