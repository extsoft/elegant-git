package git

import (
	"fmt"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/legacy"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/text"
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
				return shared.WithLock(func() error {
					return configureRun(cmd)
				})
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

// AutoConfigure runs configure when Git is not yet acquired, this is not
// `git configure` (or acquire-git), and the session is interactive.
func AutoConfigure(cmd *cobra.Command) error {
	return shared.WithLock(func() error {
		if config.IsGitAcquired() {
			return nil
		}
		if isGitConfigureCommand(cmd) {
			return nil
		}
		if prompt.NonInteractive(prompt.FromContext(cmd.Context())) {
			return nil
		}
		return configureRun(cmd)
	})
}

func isGitConfigureCommand(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	if cmd.Name() == "configure" && cmd.Parent() != nil && cmd.Parent().Name() == "git" {
		return true
	}
	path, ok := legacy.LegacyToPath[cmd.Name()]
	return ok && len(path) == 2 && path[0] == "git" && path[1] == "configure"
}

func configureRun(cmd *cobra.Command) error {
	p := prompt.FromContext(cmd.Context())
	printConfigurePlan()
	if err := waitToContinue(p); err != nil {
		return err
	}
	if err := config.BasicsConfiguration("--global", true, p); err != nil {
		return err
	}
	if git.ConfigGlobalGet("user.name") == "" || git.ConfigGlobalGet("user.email") == "" {
		return fmt.Errorf("user.name and user.email are required to finish global Git configuration")
	}
	if err := offerCreateWorkspaceFromGlobal(cmd); err != nil {
		return err
	}
	if err := config.StandardsConfiguration("--global"); err != nil {
		return err
	}
	if err := config.ObsoleteConfigurationsRemoving("--global"); err != nil {
		return err
	}
	if err := config.AliasesConfiguration("--global"); err != nil {
		return err
	}
	if err := config.MarkAcquired("--global"); err != nil {
		return err
	}
	text.Complete("Global Git configuration complete...")
	return nil
}

func waitToContinue(p prompt.Prompter) error {
	if prompt.NonInteractive(p) {
		return nil
	}
	text.QuestionText("Press enter to continue...")
	if t, ok := p.(*prompt.TTY); ok {
		if err := t.ReadLine(); err != nil {
			return err
		}
	}
	text.PlainText("")
	return nil
}

func printConfigurePlan() {
	text.InfoBox("Starting global Git configuration...")
	text.PlainText("This will configure your Git installation as follows:")
	text.PlainText("1. Git basics - user identity")
	text.PlainText("2. Git standards - Git options Elegant Git needs")
	text.PlainText("3. Git aliases - shortcuts to call Elegant Git through Git like `git elegant ...`")
	text.PlainText("")
	text.PlainText("Read " + cliruntime.SiteURL + "/reference/configuration/ to find out more.")
}
