package hook

import (
	"github.com/extsoft/elegant-git/internal/runtime"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newMigrateCommand() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{
		Use:   "migrate",
		Short: "Migrates repo-tracked hooks to the new layout",
		Long:  "Moves .workflows/* to .config/elegant-git/hooks/ (canonical hook names plus any other files), then removes .workflows.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return migrateCommon(dryRun)
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print planned changes without applying")
	return c
}

func migrateCommon(dryRun bool) error {
	text.InfoBox("Migrating common hooks...")
	ws := runtime.RepoLayout{RepoRoot: "."}
	newPaths, oldPaths, err := MigrateHooks(ws, false, dryRun)
	if err != nil {
		return err
	}
	if !dryRun {
		text.SuggestGitAddCommit(newPaths, oldPaths, "Migrate Elegant Git hooks")
	}
	return nil
}
