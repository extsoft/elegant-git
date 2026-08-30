// Package migrate applies silent automatic migrations of Elegant Git-owned state.
//
// Global steps are gated by Generation in shared memory (non-integer leftovers count as 0):
//   - elegant-git.acquired → acquired_version (save before unset)
//   - alias.<legacy> = "elegant …" → "!eg <object> <action>" (--global)
//
// Local steps run in every git repository, even after the global watermark is current:
//   - the same alias rewrite (--local)
//   - elegant-git.default-branch / protected-branches → per-repo memory
//   - unset local elegant-git.acquired
//
// Memory schema v1→v2 (profiles → workspaces) is applied by shared.Load, not here.
// Assisted repairs live in internal/doctor (GitInstall, CurrentRepo).
package migrate

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/legacy"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/version"
)

const depthEnv = "ELEGANT_GIT_DEPTH"

// Generation is the automatic-migration step watermark in shared memory.
// Bump when adding a new global Auto step so already-stamped installs re-run.
const Generation = 1

// Auto runs pending automatic migrations. It never fails the host command:
// errors are printed to stderr and retried on the next invocation.
func Auto() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "elegant-git: auto-migration incomplete: %s\n", err)
	}
}

func run() error {
	if nested() {
		return nil
	}
	if err := migrateRepo(); err != nil {
		return err
	}
	return migrateGlobal()
}

func nested() bool {
	d, err := strconv.Atoi(os.Getenv(depthEnv))
	if err != nil {
		return false
	}
	return d > 0
}

func migrateRepo() error {
	if _, err := memrepo.GitDir(); err != nil {
		return nil
	}
	if err := rewriteElegantAliases("--local"); err != nil {
		return err
	}
	if err := migrateBranchKeys(); err != nil {
		return err
	}
	return quietUnset("--local", config.AcquiredKey)
}

func migrateGlobal() error {
	s, err := shared.Load()
	if err != nil {
		return err
	}
	if s.MigrationsVersion >= Generation {
		return nil
	}
	imported, err := importAcquired(s)
	if err != nil {
		return err
	}
	if err := rewriteElegantAliases("--global"); err != nil {
		return err
	}
	if imported {
		if err := shared.Save(s); err != nil {
			return err
		}
	}
	if err := quietUnset("--global", config.AcquiredKey); err != nil {
		return err
	}
	s.MigrationsVersion = Generation
	return shared.Save(s)
}

func importAcquired(s *shared.State) (bool, error) {
	val := strings.TrimSpace(git.ConfigGlobalGet(config.AcquiredKey))
	if val == "" {
		return false, nil
	}
	if shared.Acquired(s) != "" {
		return false, nil
	}
	if val == config.AcquiredValueLegacy {
		val = version.Version
	}
	shared.SetAcquired(s, val)
	return true, nil
}

func migrateBranchKeys() error {
	gitDir, err := memrepo.GitDir()
	if err != nil {
		return nil
	}
	def, prot := memrepo.ReadLegacyElegantGitSettings()
	if def == "" && len(prot) == 0 {
		return nil
	}
	perRepo, err := memrepo.Load(gitDir)
	if err != nil {
		return err
	}
	if def != "" {
		perRepo.DefaultBranch = def
	}
	if len(prot) > 0 {
		perRepo.ProtectedBranches = prot
	}
	if err := memrepo.Save(gitDir, perRepo); err != nil {
		return err
	}
	for _, key := range []string{"elegant-git.default-branch", "elegant-git.protected-branches"} {
		if err := quietUnset("--local", key); err != nil {
			return err
		}
	}
	return nil
}

func rewriteElegantAliases(scope string) error {
	out := git.OutputOK("config", scope, "--get-regexp", `^alias\.`)
	if out == "" {
		return nil
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := fields[0]
		cur := strings.Join(fields[1:], " ")
		if !strings.HasPrefix(cur, "elegant ") {
			continue
		}
		name := strings.TrimPrefix(key, "alias.")
		if name == "show-commands" {
			continue
		}
		newVal := legacy.AliasValue(name)
		if newVal == "" || cur == newVal {
			continue
		}
		if err := quietSet(scope, key, newVal); err != nil {
			return err
		}
	}
	return nil
}

func quietSet(scope, key, value string) error {
	_, err := git.Output("config", scope, key, value)
	return err
}

func quietUnset(scope, key string) error {
	if git.ConfigGet(scope, key) == "" {
		return nil
	}
	_, err := git.Output("config", scope, "--unset", key)
	return err
}
