package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newAcquireGitCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWithWorkflows(cmd, acquireGitRun)
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func acquireGitRun() error {
	reader := bufio.NewReader(os.Stdin)
	if err := config.ObsoleteConfigurationsRemoving("--global"); err != nil {
		return err
	}
	if !config.IsGitAcquired() {
		text.InfoBox("Thank you for installing Elegant Git! Let's configure it...")
		fmt.Fprint(os.Stdout, installMessage())
		text.QuestionText("Would you like to apply a global configuration? (y/n) ")
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(line)) != "y" {
			fmt.Fprint(os.Stdout, localOnlyMessage())
			os.Exit(0)
		}
		text.InfoText("Applying global configuration...")
	}
	if err := config.BasicsConfiguration("--global", true, reader); err != nil {
		return err
	}
	if err := config.StandardsConfiguration("--global"); err != nil {
		return err
	}
	if err := config.MarkAcquired("--global"); err != nil {
		return err
	}
	return config.AliasesConfiguration("--global", CommandNames()...)
}

func installMessage() string {
	return fmt.Sprintf(`Elegant Git aims to standardize how a work environment should be configured.
It is achieved by

1. applying the configuration for a concrete repository only (a local
   configuration that is managed by "git elegant acquire-repository")
2. applying the configuration for both Git installation (global configuration
   that is managed by this command) and a repository (managed by
   "git elegant acquire-repository")

The second option is preferred in case of the installation of a newer Elegant
Git version as it allows you don't refresh a configuration for each local
repository where Elegant Git is used. Also, if the global configuration is
applied, it doesn't force you to use Elegant Git for all local repositories you
interact with. It is still up to you.

If needed, please read more about the configuration approach to be used on
%s/en/latest/configuration/
`, siteURL)
}

func localOnlyMessage() string {
	return `
You've decided to stay with local configurations. Great!
Now you have to follow some rules:

1. if you want to acquire existing local repository
        git-elegant acquire-repository

2. if you need to clone a repository
        git-elegant clone-repository

3. if you need to create a new repository
        git-elegant init-repository

`
}
