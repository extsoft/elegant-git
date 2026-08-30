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

func runList(t *testing.T, args ...string) (string, error) {
	t.Helper()
	if args == nil {
		args = []string{}
	}
	cmd := newListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	cmd.SetContext(context.Background())
	err := cmd.Execute()
	return buf.String(), err
}

func TestListDefaultOutsideGitListsAll(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	git.Use(git.NewMemoryRunner())
	s, _ := shared.Load()
	s.Workspaces["id-a"] = &shared.Workspace{Name: "alpha", UserName: "A", UserEmail: "a@x.com"}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	out, err := runList(t)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "alpha\tA <a@x.com>") {
		t.Fatalf("got:\n%s", out)
	}
	if strings.Contains(out, "linked workspace") || strings.Contains(out, "name:") {
		t.Fatalf("expected catalog, got:\n%s", out)
	}
}

func TestListDefaultInsideGitShowsCurrent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	gitDir := filepath.Join(dir, "r", ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	git.Use(m)

	out, err := runList(t)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "linked workspace: (not set") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestListAllInsideGitShowsCatalog(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	gitDir := filepath.Join(dir, "r", ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	git.Use(m)
	s, _ := shared.Load()
	s.Workspaces["id-a"] = &shared.Workspace{Name: "alpha", UserName: "A", UserEmail: "a@x.com"}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	out, err := runList(t, "all")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "alpha\tA <a@x.com>") {
		t.Fatalf("got:\n%s", out)
	}
	if strings.Contains(out, "linked workspace") {
		t.Fatalf("expected catalog, got:\n%s", out)
	}
}

func TestListCurrentOutsideGit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	git.Use(git.NewMemoryRunner())

	out, err := runList(t, "current")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "not inside a git work tree") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestListNamedDetails(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	git.Use(git.NewMemoryRunner())
	s := seedState(t, "id-1", "work", "Worker", "w@x.com", "repo-1", "myrepo", "/r")
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	out, err := runList(t, "work")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"name:         work", "user.name:    Worker"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestListCurrentWithUnreadableState(t *testing.T) {
	dir := t.TempDir()
	state := filepath.Join(dir, "state.json")
	if err := os.WriteFile(state, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ELEGANT_GIT_STATE_FILE", state)
	git.Use(git.NewMemoryRunner())

	out, err := runList(t, "current")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "not inside a git work tree") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestListFormatJSONInsideGitListsAll(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	gitDir := filepath.Join(dir, "r", ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	git.Use(m)
	s, _ := shared.Load()
	s.Workspaces["id-a"] = &shared.Workspace{Name: "alpha", UserName: "A", UserEmail: "a@x.com"}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	out, err := runList(t, "--format=json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"name": "alpha"`) {
		t.Fatalf("got:\n%s", out)
	}
	if strings.Contains(out, "linked workspace") {
		t.Fatalf("expected catalog, got:\n%s", out)
	}
}

func TestListCurrentRejectsFormat(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	git.Use(git.NewMemoryRunner())

	_, err := runList(t, "--format=json", "current")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--format") {
		t.Fatalf("got %v", err)
	}
}
