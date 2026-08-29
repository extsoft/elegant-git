package repo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/shared"
)

func TestSyncRegistryPathUpdatesMovedRepo(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")
	t.Setenv("ELEGANT_GIT_STATE_FILE", stateFile)

	oldPath := filepath.Join(dir, "old")
	newPath := filepath.Join(dir, "new")
	if err := os.MkdirAll(newPath, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(newPath)

	repoID := "repo-1"
	s, _ := shared.Load()
	profID, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "p", UserName: "u", UserEmail: "u@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: repoID, Name: "demo", WorkspaceID: profID, CurrentPath: oldPath,
	}); err != nil {
		t.Fatal(err)
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Repo.LocalConfig["elegant-git.repo-id"] = repoID
	git.Use(m)

	changed, absPath, err := syncRegistryPath(newPath, false)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || absPath != newPath {
		t.Fatalf("changed=%v absPath=%q", changed, absPath)
	}
	s2, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	repo, err := shared.GetRepo(s2, repoID)
	if err != nil {
		t.Fatal(err)
	}
	if repo.CurrentPath != newPath {
		t.Fatalf("current=%q", repo.CurrentPath)
	}
	if len(repo.PathHistory) != 1 || repo.PathHistory[0] != oldPath {
		t.Fatalf("history=%v", repo.PathHistory)
	}
}

func TestSyncRegistryPathSkipsUnregisteredRepo(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	t.Chdir(dir)

	m := git.NewMemoryRunner()
	m.Repo.LocalConfig["elegant-git.repo-id"] = "missing"
	git.Use(m)

	changed, _, err := syncRegistryPath(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected no update")
	}
}

func TestSyncRegistryPathRequireRegistered(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	t.Chdir(dir)

	if _, _, err := syncRegistryPath(dir, true); err == nil {
		t.Fatal("expected error without repo-id")
	}

	m := git.NewMemoryRunner()
	m.Repo.LocalConfig["elegant-git.repo-id"] = "missing"
	git.Use(m)

	if _, _, err := syncRegistryPath(dir, true); err == nil {
		t.Fatal("expected error for unknown registry entry")
	}
}

func TestSyncRegistryPathUsesCWDWhenPathEmpty(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")
	t.Setenv("ELEGANT_GIT_STATE_FILE", stateFile)
	t.Chdir(dir)

	repoID := "repo-1"
	s, _ := shared.Load()
	profID, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "p", UserName: "u", UserEmail: "u@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	oldPath := filepath.Join(dir, "elsewhere")
	if err := shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: repoID, Name: "demo", WorkspaceID: profID, CurrentPath: oldPath,
	}); err != nil {
		t.Fatal(err)
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Repo.LocalConfig["elegant-git.repo-id"] = repoID
	git.Use(m)

	changed, absPath, err := syncRegistryPath("", false)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || absPath != dir {
		t.Fatalf("changed=%v absPath=%q", changed, absPath)
	}
}
