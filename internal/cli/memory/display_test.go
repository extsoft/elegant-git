package memory

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bees-hive/elegant-git/internal/git"
	memrepo "github.com/bees-hive/elegant-git/internal/memory/repo"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
)

func TestPrintMemorySummaryNotInGitTree(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	git.Use(git.NewMemoryRunner())

	var buf bytes.Buffer
	if err := PrintMemorySummary(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"version:",
		"shared memory:",
		"workspaces: 0",
		"repositories: 0",
		"not inside a git work tree",
		"git elegant git status",
		"memory workspaces",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestPrintMemorySummaryInGitNotConfigured(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	m := git.NewMemoryRunner()
	gitDir := filepath.Join(dir, ".git")
	m.Outputs["rev-parse --git-dir"] = gitDir
	git.Use(m)

	var buf bytes.Buffer
	if err := PrintMemorySummary(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "not configured; run repo configure") {
		t.Fatalf("got:\n%s", buf.String())
	}
}

func TestPrintMemorySummaryRegisteredRepo(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")
	t.Setenv("ELEGANT_GIT_STATE_FILE", stateFile)
	gitDir := filepath.Join(dir, "proj", ".git")
	projDir := filepath.Join(dir, "proj")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	m.Repo.LocalConfig["elegant-git.repo-id"] = "repo-1"
	git.Use(m)

	s := seedState(t, "prof-1", "alice", "Alice", "a@example.com", "repo-1", "myrepo", projDir)
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := PrintMemorySummary(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "repository: myrepo ("+projDir+")") {
		t.Fatalf("got:\n%s", buf.String())
	}
}

func TestPrintGitStatusNoProfileDump(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	m := git.NewMemoryRunner()
	m.GlobalConfig["user.name"] = "Global"
	git.Use(m)
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	shared.SetAcquired(s, "true")
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	s.Workspaces["p1"] = &shared.Workspace{Name: "work", UserName: "W", UserEmail: "w@x.com", LinkedRepos: []string{}}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := PrintGitStatus(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"version:", "shared memory:", "global git identity:", "elegant-git.acquired: true"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(out, "profiles:\n") {
		t.Fatal("should not dump full workspaces section")
	}
}

func TestPrintRepoStatusNotInGitTree(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	git.Use(git.NewMemoryRunner())

	var buf bytes.Buffer
	if err := PrintRepoStatus(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "not inside a git work tree") {
		t.Fatalf("got:\n%s", buf.String())
	}
}

func TestPrintRepoStatusLinkedProfileNameOnly(t *testing.T) {
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

	s := seedState(t, "prof-1", "work", "Worker", "w@x.com", "repo-1", "myrepo", repoRoot)
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	if err := memrepo.Save(gitDir, &memrepo.State{
		DefaultBranch:     "main",
		ProtectedBranches: []string{"main"},
	}); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := PrintRepoStatus(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "linked workspace: work") {
		t.Errorf("missing linked workspace name: %s", out)
	}
	if strings.Contains(out, "user.name:    Worker") {
		t.Error("should not print full workspace fields in repo status")
	}
	for _, want := range []string{"default branch: main", "protected branches:", "local git identity:"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestReposWithProfile(t *testing.T) {
	s := &shared.State{
		Repositories: map[string]*shared.Repository{
			"a": {WorkspaceID: "p"},
			"b": {WorkspaceID: ""},
			"c": nil,
		},
	}
	if n := reposWithWorkspace(s); n != 1 {
		t.Fatalf("got %d want 1", n)
	}
}

func TestFileStatusLine(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "nope.json")
	if !strings.Contains(fileStatusLine(missing), "not created yet") {
		t.Fatal(missing)
	}
	exists := filepath.Join(dir, "exists.json")
	if err := os.WriteFile(exists, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if fileStatusLine(exists) != exists {
		t.Fatalf("got %q", fileStatusLine(exists))
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
