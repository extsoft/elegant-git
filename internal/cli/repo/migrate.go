package repo

import (
	"fmt"
	"os"
	"strings"

	hookcmd "github.com/bees-hive/elegant-git/internal/cli/hook"
	"github.com/bees-hive/elegant-git/internal/cli/legacy"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/deprecation"
	"github.com/bees-hive/elegant-git/internal/git"
	memrepo "github.com/bees-hive/elegant-git/internal/memory/repo"
	"github.com/bees-hive/elegant-git/internal/memory/repoid"
	"github.com/bees-hive/elegant-git/internal/runtime"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newMigrateCommand() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate local aliases, hooks, and pipe keys",
		Long:  "Rewrites local git aliases, moves personal hooks, and renames pipe config keys.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return migrateLocal(dryRun)
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print planned changes without applying")
	return c
}

func migrateLocal(dryRun bool) error {
	text.InfoBox("Migrating local Elegant Git configuration...")
	globalAcquired := config.IsGitAcquired()
	if !globalAcquired {
		for _, legacyName := range legacy.LegacyNames() {
			if legacyName == "show-commands" {
				continue
			}
			newVal := legacy.AliasValue(legacyName)
			cur, _ := git.Output("config", "--local", "--get", "alias."+legacyName)
			cur = strings.TrimSpace(cur)
			if cur != newVal {
				fmt.Fprintf(os.Stdout, "  alias.%s -> %q\n", legacyName, newVal)
				if !dryRun {
					_ = git.Verbose("config", "--local", "alias."+legacyName, newVal)
				}
			}
		}
	} else {
		text.InfoText("Removing redundant local configuration (global Elegant Git configuration is applied).")
		if err := config.CleanupRedundantLocalInstall(dryRun); err != nil {
			return err
		}
	}
	ws := runtime.Workspace{RepoRoot: "."}
	newPaths, oldPaths, err := hookcmd.MigrateHooks(ws, true, dryRun)
	if err != nil {
		return err
	}
	if !globalAcquired {
		for legacyName, id := range legacy.LegacyToID {
			migratePipeKey(legacyName, id, "stash", dryRun)
			migratePipeKey(legacyName, id, "current-branch", dryRun)
		}
	}
	if !dryRun {
		if !globalAcquired {
			if err := config.MigrateAcquiredValue("--local"); err != nil {
				return err
			}
		}
		if err := migrateRepoMemory(); err != nil {
			return err
		}
		text.SuggestGitAddCommit(newPaths, oldPaths, "Migrate Elegant Git hooks")
	}
	if dryRun {
		text.Complete("Local migration complete (dry run).")
	} else {
		text.Complete("Local migration complete.")
	}
	return nil
}

func migrateRepoMemory() error {
	_, err := repoid.EnsureLocal()
	if err != nil {
		return err
	}
	gitDir, err := memrepo.GitDir()
	if err != nil {
		return err
	}
	perRepo, err := memrepo.Load(gitDir)
	if err != nil {
		return err
	}
	def, prot := memrepo.ReadLegacyElegantGitSettings()
	if def != "" {
		perRepo.DefaultBranch = def
	}
	if len(prot) > 0 {
		perRepo.ProtectedBranches = prot
	}
	if err := memrepo.UnsetLegacyElegantGitKeys(); err != nil {
		return err
	}
	return memrepo.Save(gitDir, perRepo)
}

func migratePipeKey(legacyName string, id cmdid.ID, suffix string, dryRun bool) {
	oldKey := "elegant." + legacyName + "-" + suffix
	newKey := id.ConfigKeySuffix(suffix)
	oldVal, err := git.Output("config", "--local", "--get", oldKey)
	if err != nil || strings.TrimSpace(oldVal) == "" {
		return
	}
	fmt.Fprintf(os.Stdout, "  config %s -> %s\n", oldKey, newKey)
	if !dryRun {
		deprecation.RecordLegacyPipeKey(oldKey)
		_ = git.Verbose("config", "--local", newKey, strings.TrimSpace(oldVal))
		_ = git.ConfigLocalUnset(oldKey)
	}
}
