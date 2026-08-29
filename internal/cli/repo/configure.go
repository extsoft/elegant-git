package repo

import (
	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var configureID = cmdid.ID{Command: "repo", Action: "configure"}

func workspaceInput(index int, workspace *string) argspec.Input {
	return argspec.PositionalInputWithComplete("workspace", index, true, "Workspace name", workspace, nil, sources.WorkspacesWithCreateNew, true)
}

func configureSpec(workspace *string) argspec.Spec {
	return argspec.Spec{Inputs: []argspec.Input{
		workspaceInput(0, workspace),
	}}
}

func newConfigureCommand() *cobra.Command {
	var workspaceName string
	spec := configureSpec(&workspaceName)
	c := &cobra.Command{
		Use:   "configure <workspace>",
		Short: "Configures the current local Git repository",
		Long:  "Applies local Elegant Git configuration for this repository.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, configureID, func() error {
				if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
					return err
				}
				return configureRun(cmd, workspaceName)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func configureRun(cmd *cobra.Command, workspaceName string) error {
	p := prompt.FromContext(cmd.Context())
	if updated, path, err := syncRegistryPath("", false); err != nil {
		return err
	} else if updated {
		text.InfoText("Repository path updated to " + path)
	}
	if err := configureLocalGitInstallPre(); err != nil {
		return err
	}
	assigned, err := configureWithMemory(cmd, workspaceName)
	if err != nil {
		return err
	}
	if err := configureLocalGitInstallPost(); err != nil {
		return err
	}
	if !assigned {
		if err := config.ConfigureSignature(p); err != nil {
			return err
		}
	}
	text.Complete("Repository configuration complete.")
	return nil
}

func configureLocalGitInstallPre() error {
	if config.NeedsLocalGitInstall() {
		return config.ObsoleteConfigurationsRemoving("--local")
	}
	return config.CleanupRedundantLocalInstall(false)
}

func configureLocalGitInstallPost() error {
	if !config.NeedsLocalGitInstall() {
		return nil
	}
	if err := config.StandardsConfiguration("--local"); err != nil {
		return err
	}
	return config.AliasesConfiguration("--local")
}

// ConfigureRun is exported for clone/init to call configure logic.
func ConfigureRun(cmd *cobra.Command, workspace string) error {
	return configureRun(cmd, workspace)
}
