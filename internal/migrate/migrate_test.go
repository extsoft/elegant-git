package migrate

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/version"
)

func TestAutoShortCircuitsGlobalOnWatermark(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	t.Setenv(depthEnv, "")
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	s.MigrationsVersion = Generation
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	m := git.NewMemoryRunner()
	m.GlobalConfig["alias.start-work"] = "elegant start-work"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	Auto()
	if m.GlobalConfig["alias.start-work"] != "elegant start-work" {
		t.Fatal("watermark match must skip global alias rewrite")
	}
}

func TestAutoMigratesLocalStateAfterGlobalWatermark(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(dir, "repo-state.json"))
	t.Setenv(depthEnv, "")
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	s.MigrationsVersion = Generation
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(dir, ".git")
	m.GlobalConfig["alias.start-work"] = "elegant start-work"
	m.Repo.LocalConfig["alias.accept-work"] = "elegant accept-work"
	m.Repo.LocalConfig["elegant-git.default-branch"] = "develop"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	Auto()
	if m.GlobalConfig["alias.start-work"] != "elegant start-work" {
		t.Fatal("global watermark must skip global alias rewrite")
	}
	if m.Repo.LocalConfig["alias.accept-work"] != "!eg work accept" {
		t.Fatalf("local alias = %q", m.Repo.LocalConfig["alias.accept-work"])
	}
	per, err := memrepo.Load(filepath.Join(dir, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	if per.DefaultBranch != "develop" {
		t.Fatalf("default_branch = %q", per.DefaultBranch)
	}
}

func TestAutoSkipsNestedInvocation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	t.Setenv(depthEnv, "1")
	m := git.NewMemoryRunner()
	m.GlobalConfig["alias.start-work"] = "elegant start-work"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	Auto()
	if m.GlobalConfig["alias.start-work"] != "elegant start-work" {
		t.Fatal("nested invocation must skip")
	}
}

func TestAutoRewritesElegantAliasesOnly(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	t.Setenv(depthEnv, "")
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(dir, ".git")
	m.GlobalConfig["alias.start-work"] = "elegant start-work"
	m.GlobalConfig["alias.save-work"] = "!eg work save"
	m.Repo.LocalConfig["alias.accept-work"] = "elegant accept-work"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	Auto()
	if m.GlobalConfig["alias.start-work"] != "!eg work start" {
		t.Fatalf("start-work = %q", m.GlobalConfig["alias.start-work"])
	}
	if m.GlobalConfig["alias.save-work"] != "!eg work save" {
		t.Fatalf("save-work = %q", m.GlobalConfig["alias.save-work"])
	}
	if _, ok := m.GlobalConfig["alias.amend-work"]; ok {
		t.Fatal("must not create missing aliases")
	}
	if m.Repo.LocalConfig["alias.accept-work"] != "!eg work accept" {
		t.Fatalf("accept-work = %q", m.Repo.LocalConfig["alias.accept-work"])
	}
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	if s.MigrationsVersion != Generation {
		t.Fatalf("watermark = %d", s.MigrationsVersion)
	}
}

func TestAutoImportsAcquiredMarker(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	t.Setenv(depthEnv, "")
	m := git.NewMemoryRunner()
	m.GlobalConfig[config.AcquiredKey] = config.AcquiredValueLegacy
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	Auto()
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	if shared.Acquired(s) != version.Version {
		t.Fatalf("acquired = %q", shared.Acquired(s))
	}
	if _, ok := m.GlobalConfig[config.AcquiredKey]; ok {
		t.Fatal("legacy acquired key should be unset")
	}
	if s.MigrationsVersion != Generation {
		t.Fatalf("watermark = %d", s.MigrationsVersion)
	}
}

func TestAutoMovesLegacyBranchKeys(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(dir, "repo-state.json"))
	t.Setenv(depthEnv, "")
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(dir, ".git")
	m.Repo.LocalConfig["elegant-git.default-branch"] = "develop"
	m.Repo.LocalConfig["elegant-git.protected-branches"] = "develop main"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	Auto()
	per, err := memrepo.Load(filepath.Join(dir, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	if per.DefaultBranch != "develop" {
		t.Fatalf("default_branch = %q", per.DefaultBranch)
	}
	if _, ok := m.Repo.LocalConfig["elegant-git.default-branch"]; ok {
		t.Fatal("legacy default-branch should be unset")
	}
}

func TestAutoFailureDoesNotStampOrReturn(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	t.Setenv(depthEnv, "")
	m := git.NewMemoryRunner()
	m.GlobalConfig["alias.start-work"] = "elegant start-work"
	m.FailOn["config --global alias.start-work !eg work start"] = errors.New("boom")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	Auto()
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	if s.MigrationsVersion != 0 {
		t.Fatalf("watermark stamped after failure: %d", s.MigrationsVersion)
	}
}

func TestAutoDoesNotStampWhenUnsetAcquiredFails(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	t.Setenv(depthEnv, "")
	m := git.NewMemoryRunner()
	m.GlobalConfig[config.AcquiredKey] = config.AcquiredValueLegacy
	m.FailOn["config --global --unset "+config.AcquiredKey] = errors.New("boom")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	Auto()
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	if s.MigrationsVersion != 0 {
		t.Fatalf("watermark stamped after unset failure: %d", s.MigrationsVersion)
	}
	if shared.Acquired(s) != version.Version {
		t.Fatalf("acquired should be saved before unset; got %q", shared.Acquired(s))
	}
}
