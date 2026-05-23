package repo

import (
	"bufio"

	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/spf13/cobra"
)

var configureID = cmdid.ID{Command: "repo", Action: "configure"}

func newConfigureCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "configure",
		Short: "Configures the current local Git repository",
		Long:  "Applies local Elegant Git configuration for this repository.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cliruntime.RunWithWorkflows(cmd, configureID, func() error {
				return configureRun(cmd)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func configureRun(cmd *cobra.Command) error {
	reader := bufio.NewReader(cliruntime.StdinFromContext(cmd.Context()))
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
		if err := config.AliasesConfiguration("--local"); err != nil {
			return err
		}
	}
	return config.ConfigureSignature(reader)
}

// ConfigureRun is exported for clone/init to call acquire-repository logic.
func ConfigureRun(cmd *cobra.Command) error {
	return configureRun(cmd)
}
