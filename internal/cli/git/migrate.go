package git

import (
	"fmt"
	"os"
	"strings"

	"github.com/extsoft/elegant-git/internal/cli/legacy"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newMigrateCommand() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate global aliases and acquired marker",
		Long:  "Rewrites legacy git aliases to the new object-first form and migrates elegant-git.acquired into shared memory.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return migrateGlobal(dryRun)
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print planned changes without applying")
	return c
}

func migrateGlobal(dryRun bool) error {
	text.InfoBox("Migrating global Elegant Git configuration...")
	for _, legacyName := range legacy.LegacyNames() {
		if legacyName == "show-commands" {
			continue
		}
		newVal := legacy.AliasValue(legacyName)
		oldVal := "elegant " + legacyName
		cur, _ := git.Output("config", "--global", "--get", "alias."+legacyName)
		cur = strings.TrimSpace(cur)
		if cur == newVal {
			continue
		}
		fmt.Fprintf(os.Stdout, "  alias.%s: %q -> %q\n", legacyName, cur, newVal)
		if !dryRun && (cur == oldVal || cur == newVal || cur != "") {
			if err := git.Verbose("config", "--global", "alias."+legacyName, newVal); err != nil {
				return err
			}
		}
	}
	if err := config.MigrateAcquiredValue("--global"); err != nil {
		return err
	}
	if !dryRun {
		if err := migrateSharedMemorySchema(); err != nil {
			return err
		}
	}
	if dryRun {
		text.Complete("Global migration complete (dry run).")
	} else {
		text.Complete("Global migration complete.")
	}
	return nil
}

func migrateSharedMemorySchema() error {
	s, err := shared.Load()
	if err != nil {
		return err
	}
	if !shared.LastLoadHadLegacyKeys() {
		return nil
	}
	fmt.Fprintln(os.Stdout, "  shared memory: rewriting profiles/profile_id -> workspaces/workspace_id")
	return shared.Save(s)
}
