package repo

import (
	"fmt"
	"os"
	"strings"

	hookcmd "github.com/extsoft/elegant-git/internal/cli/hook"
	"github.com/extsoft/elegant-git/internal/cli/legacy"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/memory/repoid"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/runtime"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newMigrateCommand() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate local aliases and hooks",
		Long:  "Rewrites local git aliases and moves personal hooks.",
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
		if err := config.EnsureElegantAlias("--local", dryRun); err != nil {
			return err
		}
		for _, legacyName := range legacy.LegacyNames() {
			if legacyName == "show-commands" {
				continue
			}
			newVal := legacy.AliasValue(legacyName)
			cur, _ := git.Output("config", "--local", "--get", "alias."+legacyName)
			cur = strings.TrimSpace(cur)
			if cur != newVal {
				config.RecordStaleAlias(cur, newVal)
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
	ws := runtime.RepoLayout{RepoRoot: "."}
	newPaths, oldPaths, err := hookcmd.MigrateHooks(ws, true, dryRun)
	if err != nil {
		return err
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
		if err := migrateSharedMemorySchema(); err != nil {
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
	// Always save so v1 profile_id is rewritten to workspace_id when present.
	return memrepo.Save(gitDir, perRepo)
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
