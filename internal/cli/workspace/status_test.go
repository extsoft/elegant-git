package workspace

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/shared"
)

func TestStatusCommandOutsideGit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	git.Use(git.NewMemoryRunner())

	cmd := newStatusCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetContext(context.Background())
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "not inside a git work tree") {
		t.Fatalf("got:\n%s", buf.String())
	}
}

func TestStatusCommandNoRepoID(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	gitDir := filepath.Join(dir, "r", ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	git.Use(m)

	cmd := newStatusCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetContext(context.Background())
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "linked workspace: (not set") {
		t.Fatalf("got:\n%s", buf.String())
	}
}

func TestStatusCommandLinkedProfile(t *testing.T) {
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
	s.Workspaces["prof-1"] = &shared.Workspace{Name: "work", UserName: "Worker", UserEmail: "w@x.com", LinkedRepos: []string{"repo-1"}}
	s.Repositories["repo-1"] = &shared.Repository{Name: "myrepo", WorkspaceID: "prof-1", CurrentPath: repoRoot}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := newStatusCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetContext(context.Background())
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"name:", "work", "user.name:", "Worker", "Further steps:"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}
