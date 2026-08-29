package repo

import (
	"path/filepath"
	"testing"

	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/shared"
)

func TestMigrateLocalRemovesAliasesWhenGlobalAcquired(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, _ := shared.Load()
	shared.SetAcquired(s, "1.0.0")
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	m := git.NewMemoryRunner()
	m.Outputs["config --local --get-regexp "+config.AcquiredKey] = config.AcquiredKey + " true"
	m.Outputs["config --local --get-regexp ^alias\\."] = "alias.start-work elegant work start"
	git.Use(m)

	if err := migrateLocal(true); err != nil {
		t.Fatal(err)
	}
	for _, c := range m.Calls {
		if len(c.Args) >= 4 && c.Args[0] == "config" && c.Args[2] == "alias.start-work" {
			t.Fatalf("unexpected alias rewrite: %+v", c.Args)
		}
	}
}

func TestMigrateLocalRewritesAliasesWhenLocalOnly(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	m := git.NewMemoryRunner()
	m.Outputs["config --local --get alias.start-work"] = "elegant start-work"
	git.Use(m)

	if err := migrateLocal(true); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range m.Calls {
		if len(c.Args) >= 4 && c.Args[1] == "--local" && c.Args[2] == "--get" && c.Args[3] == "alias.start-work" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected alias migration when global not acquired")
	}
}
