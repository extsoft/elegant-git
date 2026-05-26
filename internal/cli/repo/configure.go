package repo

import (
	"io"

	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

var configureID = cmdid.ID{Command: "repo", Action: "configure"}

func newConfigureCommand() *cobra.Command {
	var profileName string
	c := &cobra.Command{
		Use:   "configure",
		Short: "Configures the current local Git repository",
		Long:  "Applies local Elegant Git configuration for this repository.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cliruntime.RunWithWorkflows(cmd, configureID, func() error {
				return configureRun(cmd, profileName)
			})
		},
	}
	c.Flags().StringVar(&profileName, "profile", "", "profile name to use")
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func configureRun(cmd *cobra.Command, profileFlag string) error {
	p := prompt.FromContext(cmd.Context())
	reader := promptReader(cmd, p)
	if err := config.ObsoleteConfigurationsRemoving("--local"); err != nil {
		return err
	}
	if err := configureWithMemory(cmd, profileFlag); err != nil {
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

func promptReader(cmd *cobra.Command, p prompt.Prompter) io.Reader {
	return cliruntime.StdinFromContext(cmd.Context())
}

// ConfigureRun is exported for clone/init to call configure logic.
func ConfigureRun(cmd *cobra.Command) error {
	return configureRun(cmd, "")
}
