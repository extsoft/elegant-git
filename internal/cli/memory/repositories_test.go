package memory

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/memory/shared"
)

func TestListRepositories(t *testing.T) {
	s := &shared.State{
		Workspaces: map[string]*shared.Workspace{
			"p1": {Name: "work", UserName: "W", UserEmail: "w@x.com", LinkedRepos: []string{}},
		},
		Repositories: map[string]*shared.Repository{
			"r2": {Name: "beta", WorkspaceID: "p1", CurrentPath: "/b"},
			"r1": {Name: "alpha", WorkspaceID: "p1", CurrentPath: "/a"},
		},
	}
	var buf bytes.Buffer
	if err := listRepositories(&buf, s); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 || lines[0] != "alpha\twork\t/a" || lines[1] != "beta\twork\t/b" {
		t.Fatalf("got %q", lines)
	}
}

func TestShowRepositoryWithProfile(t *testing.T) {
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "proj")
	gitDir := filepath.Join(repoPath, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	s := seedState(t, "p1", "work", "Worker", "w@x.com", "r1", "proj", repoPath)
	repo := s.Repositories["r1"]
	repo.OriginURL = "https://example.com/proj.git"
	repo.PathHistory = []string{"/old"}

	var buf bytes.Buffer
	if err := showRepository(&buf, s, repo); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"name:  proj", "origin:", "previous paths:", "workspace:", "name:         work",
		"per-repo memory:",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestShowRepositoryMissingProfile(t *testing.T) {
	repo := &shared.Repository{Name: "orphan", WorkspaceID: "missing", CurrentPath: t.TempDir()}
	s := &shared.State{Workspaces: map[string]*shared.Workspace{}, Repositories: map[string]*shared.Repository{"r": repo}}
	var buf bytes.Buffer
	if err := showRepository(&buf, s, repo); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "workspace: (missing from shared memory)") {
		t.Fatalf("got:\n%s", buf.String())
	}
}
