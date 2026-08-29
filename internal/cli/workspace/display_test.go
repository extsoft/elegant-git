package workspace

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/memory/shared"
)

func TestPrintListTable(t *testing.T) {
	s := &shared.State{
		Workspaces: map[string]*shared.Workspace{
			"id-b": {Name: "beta", UserName: "B", UserEmail: "b@x.com", LinkedRepos: []string{"r1"}},
			"id-a": {Name: "alpha", UserName: "A", UserEmail: "a@x.com", LinkedRepos: []string{}},
		},
	}
	var buf bytes.Buffer
	if err := PrintList(&buf, s, "table"); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines: %q", lines)
	}
	if lines[0] != "alpha\tA <a@x.com>\t0 repo(s)" {
		t.Fatalf("first line: %q", lines[0])
	}
	if lines[1] != "beta\tB <b@x.com>\t1 repo(s)" {
		t.Fatalf("second line: %q", lines[1])
	}
}

func TestPrintListJSON(t *testing.T) {
	s := &shared.State{
		Workspaces: map[string]*shared.Workspace{
			"id-a": {Name: "alpha", UserName: "A", UserEmail: "a@x.com", LinkedRepos: []string{}},
		},
	}
	var buf bytes.Buffer
	if err := PrintList(&buf, s, "json"); err != nil {
		t.Fatal(err)
	}
	var got map[string]*shared.Workspace
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["id-a"] == nil || got["id-a"].Name != "alpha" {
		t.Fatalf("got %+v", got)
	}
}

func TestPrintDetailsTable(t *testing.T) {
	s := seedState(t, "id-1", "work", "Worker", "w@x.com", "repo-1", "myrepo", "/r")
	var buf bytes.Buffer
	if err := PrintDetails(&buf, s, "work", "table"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"name:         work", "id:           id-1", "user.name:    Worker", "namespaces:   (none)", "linked repos:"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestPrintDetailsJSON(t *testing.T) {
	s := seedState(t, "id-1", "work", "Worker", "w@x.com", "", "", "")
	var buf bytes.Buffer
	if err := PrintDetails(&buf, s, "work", "json"); err != nil {
		t.Fatal(err)
	}
	var got struct {
		ID        string            `json:"id"`
		Workspace *shared.Workspace `json:"workspace"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "id-1" || got.Workspace.Name != "work" {
		t.Fatalf("got %+v", got)
	}
}

func TestPrintDetailsNotFound(t *testing.T) {
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(t.TempDir(), "state.json"))
	s, _ := shared.Load()
	var buf bytes.Buffer
	err := PrintDetails(&buf, s, "nope", "table")
	if !errors.Is(err, shared.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestPrintWorkspaceStatusBranches(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	git.Use(git.NewMemoryRunner())

	var buf bytes.Buffer
	if err := PrintWorkspaceStatus(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "not inside a git work tree") {
		t.Fatalf("got:\n%s", buf.String())
	}

	gitDir := filepath.Join(dir, "r", ".git")
	repoRoot := filepath.Join(dir, "r")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	git.Use(m)

	buf.Reset()
	if err := PrintWorkspaceStatus(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "linked workspace: (not set") {
		t.Fatalf("got:\n%s", buf.String())
	}

	m.Repo.LocalConfig["elegant-git.repo-id"] = "repo-1"
	s := seedState(t, "prof-1", "work", "Worker", "w@x.com", "repo-1", "myrepo", repoRoot)
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	buf.Reset()
	if err := PrintWorkspaceStatus(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"name:         work", "user.name:    Worker", "namespaces:   (none)", "linked repos:"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	m.Repo.LocalConfig["elegant-git.repo-id"] = "missing"
	buf.Reset()
	if err := PrintWorkspaceStatus(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "not in registry") {
		t.Fatalf("got:\n%s", buf.String())
	}
}

func TestPrintWorkspaceStatusPerRepoOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	gitDir := filepath.Join(dir, "r", ".git")
	repoRoot := filepath.Join(dir, "r")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(gitDir, "elegant-git", "state.json"))

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	m.Repo.LocalConfig["elegant-git.repo-id"] = "repo-1"
	git.Use(m)

	s := seedState(t, "prof-1", "work", "Worker", "w@x.com", "repo-1", "myrepo", repoRoot)
	s.Workspaces["prof-2"] = &shared.Workspace{Name: "alt", UserName: "Alt", UserEmail: "a@x.com", LinkedRepos: []string{}}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	if err := memrepo.Save(gitDir, &memrepo.State{WorkspaceID: "prof-2"}); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := PrintWorkspaceStatus(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "per-repo memory workspace:") || !strings.Contains(out, "name:         alt") {
		t.Fatalf("got:\n%s", out)
	}
}

func seedState(t *testing.T, profID, profName, userName, email, repoID, repoName, repoPath string) *shared.State {
	t.Helper()
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	if profID != "" {
		s.Workspaces[profID] = &shared.Workspace{
			Name: profName, UserName: userName, UserEmail: email, LinkedRepos: []string{},
		}
		if repoID != "" {
			s.Workspaces[profID].LinkedRepos = []string{repoID}
		}
	}
	if repoID != "" {
		s.Repositories[repoID] = &shared.Repository{
			Name: repoName, WorkspaceID: profID, CurrentPath: repoPath,
		}
	}
	return s
}
