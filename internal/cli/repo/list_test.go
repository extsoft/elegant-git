package repo

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/memory/shared"
)

func TestListCommandOutsideGit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	git.Use(git.NewMemoryRunner())
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	s.Workspaces["p1"] = &shared.Workspace{Name: "work", UserName: "W", UserEmail: "w@x.com"}
	s.Repositories["r1"] = &shared.Repository{Name: "alpha", WorkspaceID: "p1", CurrentPath: "/a"}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := newListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetContext(context.Background())
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "alpha\n  workspace: work") {
		t.Fatalf("expected catalog, got:\n%s", out)
	}
	if strings.Contains(out, "not inside a git work tree") {
		t.Fatalf("empty arg outside git must list all, got:\n%s", out)
	}
}

func TestListCommandCurrentOutsideGit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	git.Use(git.NewMemoryRunner())

	cmd := newListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"current"})
	cmd.SetContext(context.Background())
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "not inside a git work tree") {
		t.Fatalf("got:\n%s", buf.String())
	}
}

func TestListCommandInsideGit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	gitDir := filepath.Join(dir, "r", ".git")
	repoRoot := filepath.Join(dir, "r")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	m.Repo.LocalConfig["elegant-git.repo-id"] = "repo-1"
	git.Use(m)

	s, _ := shared.Load()
	s.Workspaces["prof-1"] = &shared.Workspace{Name: "work", UserName: "W", UserEmail: "w@x.com", LinkedRepos: []string{"repo-1"}}
	s.Repositories["repo-1"] = &shared.Repository{Name: "myrepo", WorkspaceID: "prof-1", CurrentPath: repoRoot}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := newListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetContext(context.Background())
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "linked workspace: work") || !strings.Contains(out, "Local git identity") || !strings.Contains(out, "Further steps:") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestListCommandAll(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	git.Use(git.NewMemoryRunner())
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	s.Workspaces["p1"] = &shared.Workspace{Name: "work", UserName: "W", UserEmail: "w@x.com", LinkedRepos: []string{}}
	s.Repositories["r2"] = &shared.Repository{Name: "beta", WorkspaceID: "p1", CurrentPath: "/b"}
	s.Repositories["r1"] = &shared.Repository{Name: "alpha", WorkspaceID: "p1", CurrentPath: "/a"}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := newListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"all"})
	cmd.SetContext(context.Background())
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"alpha",
		"  workspace: work",
		"  path:      /a",
		"  explore:   eg repo list alpha",
		"",
		"beta",
		"  workspace: work",
		"  path:      /b",
		"  explore:   eg repo list beta",
		"",
	}, "\n")
	if buf.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestListCommandNamed(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	repoPath := filepath.Join(dir, "proj")
	gitDir := filepath.Join(repoPath, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	git.Use(git.NewMemoryRunner())
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	s.Workspaces["p1"] = &shared.Workspace{Name: "work", UserName: "Worker", UserEmail: "w@x.com", LinkedRepos: []string{"r1"}}
	s.Repositories["r1"] = &shared.Repository{
		Name: "proj", WorkspaceID: "p1", CurrentPath: repoPath,
		OriginURL: "https://example.com/proj.git", PathHistory: []string{"/old"},
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := newListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"proj"})
	cmd.SetContext(context.Background())
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"name:", "proj", "origin:", "Previous paths", "Workspace", "work",
		"Per-repo memory", "Further steps:",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestListCommandLinkedWorkspaceNameOnly(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")
	t.Setenv("ELEGANT_GIT_STATE_FILE", stateFile)
	gitDir := filepath.Join(dir, "r", ".git")
	repoRoot := filepath.Join(dir, "r")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	repoState := filepath.Join(gitDir, "elegant-git", "state.json")
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", repoState)

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	m.Repo.LocalConfig["elegant-git.repo-id"] = "repo-1"
	m.Repo.LocalConfig["user.name"] = "Local"
	git.Use(m)

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	s.Workspaces["prof-1"] = &shared.Workspace{Name: "work", UserName: "Worker", UserEmail: "w@x.com", LinkedRepos: []string{"repo-1"}}
	s.Repositories["repo-1"] = &shared.Repository{Name: "myrepo", WorkspaceID: "prof-1", CurrentPath: repoRoot}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	if err := memrepo.Save(gitDir, &memrepo.State{
		DefaultBranch:     "main",
		ProtectedBranches: []string{"main"},
	}); err != nil {
		t.Fatal(err)
	}

	cmd := newListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(context.Background())
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "linked workspace: work") {
		t.Errorf("missing linked workspace name: %s", out)
	}
	if strings.Contains(out, "user.name:    Worker") {
		t.Error("should not print full workspace fields in repo list")
	}
	for _, want := range []string{"default branch: main", "protected branches:", "Local git identity"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestShowRepositoryMissingWorkspace(t *testing.T) {
	repo := &shared.Repository{Name: "orphan", WorkspaceID: "missing", CurrentPath: t.TempDir()}
	s := &shared.State{Workspaces: map[string]*shared.Workspace{}, Repositories: map[string]*shared.Repository{"r": repo}}
	var buf bytes.Buffer
	if err := showRepository(&buf, s, repo, "table"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "workspace: (missing from shared memory)") {
		t.Fatalf("got:\n%s", buf.String())
	}
}

func TestListCommandAllJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	git.Use(git.NewMemoryRunner())
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	s.Repositories["r1"] = &shared.Repository{Name: "alpha", WorkspaceID: "p1", CurrentPath: "/a"}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := newListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--format=json", "all"})
	cmd.SetContext(context.Background())
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"name": "alpha"`) {
		t.Fatalf("got:\n%s", buf.String())
	}
}

func TestListCurrentRejectsFormat(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	git.Use(git.NewMemoryRunner())

	cmd := newListCommand()
	cmd.SetArgs([]string{"--format=json", "current"})
	cmd.SetContext(context.Background())
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error")
	}
}
