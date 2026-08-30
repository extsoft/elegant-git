package repo

import (
	"fmt"
	"io"

	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/extsoft/elegant-git/internal/hooks"
	"github.com/extsoft/elegant-git/internal/runtime"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newMigrateCommand() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{
		Use:    "migrate",
		Hidden: true,
		Short:  "Deprecated; migrations run automatically",
		Long:   "Hidden compatibility shim. Remaining repository issues are repaired by `eg repo doctor`.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			deprecation.Record(deprecation.DEP016, "repo migrate", "eg repo doctor", "eg repo doctor")
			fmt.Fprintln(cmd.ErrOrStderr(), "migrations now run automatically; run `eg repo doctor` for the rest")
			return migrateHookTiers(cmd.OutOrStdout(), dryRun)
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print planned hook moves without applying")
	return c
}

func migrateHookTiers(w io.Writer, dryRun bool) error {
	ws := runtime.DefaultRepoLayout()
	if _, _, err := hooks.Migrate(ws, true, dryRun, w); err != nil {
		return err
	}
	newPaths, oldPaths, err := hooks.Migrate(ws, false, dryRun, w)
	if err != nil {
		return err
	}
	if !dryRun {
		text.SuggestGitAddCommitTo(w, newPaths, oldPaths, "Migrate Elegant Git hooks")
	}
	return nil
}
