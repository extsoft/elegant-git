package repo

import (
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
)

func TestUnsetLegacyElegantGitKeys(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Repo.LocalConfig["elegant-git.default-branch"] = "main"
	m.Repo.LocalConfig["elegant-git.protected-branches"] = "main develop"
	git.Use(m)
	if err := UnsetLegacyElegantGitKeys(); err != nil {
		t.Fatal(err)
	}
	if len(m.Repo.LocalConfig) != 0 {
		t.Fatalf("expected empty local config, got %v", m.Repo.LocalConfig)
	}
}
