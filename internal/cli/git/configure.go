package git

import (
	"fmt"
	"os"

	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var configureID = cmdid.ID{Command: "git", Action: "configure"}

func newConfigureCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "configure",
		Short: "Configures your Git installation",
		Long:  "Applies global Elegant Git configuration, standards, and git aliases.",
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
	p := prompt.FromContext(cmd.Context())
	if err := config.ObsoleteConfigurationsRemoving("--global"); err != nil {
		return err
	}
	if !config.IsGitAcquired() {
		text.InfoBox("Thank you for installing Elegant Git! Let's configure it...")
		fmt.Fprint(os.Stdout, installMessage())
		ok, err := p.Confirm("Would you like to apply a global configuration?", false)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprint(os.Stdout, localOnlyMessage())
			os.Exit(0)
		}
		text.InfoText("Applying global configuration...")
	}
	if err := config.BasicsConfiguration("--global", true, p); err != nil {
		return err
	}
	if err := config.StandardsConfiguration("--global"); err != nil {
		return err
	}
	if err := config.MarkAcquired("--global"); err != nil {
		return err
	}
	if err := config.AliasesConfiguration("--global"); err != nil {
		return err
	}
	if err := offerCreateWorkspaceFromGlobal(cmd); err != nil {
		return err
	}
	text.Complete("Global Git configuration complete.")
	return nil
}

func installMessage() string {
	return fmt.Sprintf(`Elegant Git aims to standardize how a work environment should be configured.
It is achieved by

1. applying the configuration for a concrete repository only (a local
   configuration that is managed by "git elegant repo configure")
2. applying the configuration for both Git installation (global configuration
   that is managed by this command) and a repository (managed by
   "git elegant repo configure")

The second option is preferred in case of the installation of a newer Elegant
Git version as it allows you don't refresh a configuration for each local
repository where Elegant Git is used. Also, if the global configuration is
applied, it doesn't force you to use Elegant Git for all local repositories you
interact with. It is still up to you.

If needed, please read more about the configuration approach to be used on
%s/en/latest/configuration/
`, cliruntime.SiteURL)
}

func localOnlyMessage() string {
	return `
You've decided to stay with local configurations. Great!
Now you have to follow some rules:

1. if you want to acquire existing local repository
        git elegant repo configure

2. if you need to clone a repository
        git elegant repo clone

3. if you need to create a new repository
        git elegant repo init

`
}
