package legacy

import (
	"fmt"

	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/extsoft/elegant-git/internal/doctor"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

// RegisterObjectShims adds hidden git and memory command groups that delegate
// to self, workspace list, and repo list.
func RegisterObjectShims(root *cobra.Command) {
	gitCmd := hiddenObject("git")
	gitCmd.AddCommand(shim(root, "configure", "git configure", []string{"self", "configure"}))
	gitCmd.AddCommand(shim(root, "list", "git list", []string{"self", "list"}))
	gitCmd.AddCommand(shim(root, "doctor", "git doctor", []string{"self", "doctor"}))
	gitCmd.AddCommand(shim(root, "status", "git status", []string{"self", "list"}))
	gitCmd.AddCommand(newGitMigrateCommand())
	root.AddCommand(gitCmd)

	memoryCmd := hiddenObject("memory")
	memoryCmd.AddCommand(shim(root, "list", "memory list", []string{"self", "list"}))
	memoryCmd.AddCommand(newMemoryWorkspacesShim(root, "workspaces", "memory workspaces"))
	memoryCmd.AddCommand(newMemoryWorkspacesShim(root, "profiles", "memory profiles"))
	memoryCmd.AddCommand(shimDefaultArg(root, "repositories [name-or-path]", "memory repositories", []string{"repo", "list"}, shared.SelectorAll))
	memoryCmd.AddCommand(shim(root, "status", "memory status", []string{"self", "list"}))
	root.AddCommand(memoryCmd)
}

func hiddenObject(use string) *cobra.Command {
	c := &cobra.Command{
		Use:    use,
		Hidden: true,
		Short:  "Deprecated; use self",
	}
	annotate(c, use)
	return c
}

func shim(root *cobra.Command, use, surface string, path []string) *cobra.Command {
	c := &cobra.Command{
		Use:    use,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTarget(root, cmd, path, args)
		},
	}
	annotate(c, surface)
	return c
}

func shimDefaultArg(root *cobra.Command, use, surface string, path []string, defaultArg string) *cobra.Command {
	c := &cobra.Command{
		Use:    use,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				args = []string{defaultArg}
			}
			return runTarget(root, cmd, path, args)
		},
	}
	annotate(c, surface)
	return c
}

func newMemoryWorkspacesShim(root *cobra.Command, use, surface string) *cobra.Command {
	var format string
	c := &cobra.Command{
		Use:    use + " [name]",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			target, _, err := root.Find([]string{"workspace", "list"})
			if err != nil {
				return err
			}
			if err := target.Flags().Set("format", format); err != nil {
				return err
			}
			if len(args) == 0 {
				args = []string{shared.SelectorAll}
			}
			return runTarget(root, cmd, []string{"workspace", "list"}, args)
		},
	}
	c.Flags().StringVar(&format, "format", "table", "output format: table or json (default table)")
	annotate(c, surface)
	return c
}

func newGitMigrateCommand() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{
		Use:    "migrate",
		Hidden: true,
		Short:  "Deprecated; migrations run automatically",
		Long:   "Hidden compatibility shim. Remaining Git-install issues are repaired by `eg self doctor`.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			deprecation.Record(deprecation.DEP016, "git migrate", "eg self doctor", "eg self doctor")
			fmt.Fprintln(cmd.ErrOrStderr(), "migrations now run automatically; run `eg self doctor` for the rest")
			if dryRun {
				_, err := doctor.Run(cmd.OutOrStdout(), prompt.NewNonInteractive(), doctor.GitInstall())
				return err
			}
			return doctor.RepairGlobalAliases()
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "report remaining Git-install issues without applying")
	return c
}

func annotate(c *cobra.Command, surface string) {
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[deprecation.SurfaceAnnotation] = surface
}
