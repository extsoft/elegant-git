package self

import (
	"bytes"
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/shared"
)

func TestListCommandOutput(t *testing.T) {
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
	s.Workspaces["p"] = &shared.Workspace{Name: "p", UserName: "P", UserEmail: "p@x.com", LinkedRepos: []string{}}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := newListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(nil)
	cmd.SetContext(context.Background())

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"Version",
		"Shared memory",
		"workspaces:",
		"Global git identity",
		"elegant-git.acquired: true",
		"Further steps:",
		"eg workspace list all",
		"eg repo list all",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "profiles:\n") {
		t.Fatal("should not dump workspaces section")
	}
}

func TestListOmitsCurrentRepository(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(dir, ".git")
	m.Repo.LocalConfig["elegant-git.repo-id"] = "repo-1"
	git.Use(m)

	var buf bytes.Buffer
	if err := printSelfList(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "repository:") {
		t.Errorf("self list must not report the current repository:\n%s", out)
	}
	if strings.Contains(out, "this repository in detail") {
		t.Errorf("only the catalog pointer belongs here:\n%s", out)
	}
}

func TestListSeparatesSectionsWithBlankLines(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	git.Use(git.NewMemoryRunner())

	var buf bytes.Buffer
	if err := printSelfList(&buf); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(buf.String(), "\n")
	for _, heading := range []string{"Shared memory", "Global git identity", "Further steps:"} {
		i := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(l, heading) })
		if i < 1 {
			t.Fatalf("heading %q not found after a blank line in:\n%s", heading, buf.String())
		}
		if lines[i-1] != "" {
			t.Errorf("heading %q not preceded by a blank line, got %q", heading, lines[i-1])
		}
	}
}

func TestNewCommandSubcommands(t *testing.T) {
	c := NewCommand()
	var uses []string
	for _, sub := range c.Commands() {
		if sub.Hidden {
			continue
		}
		uses = append(uses, sub.Name())
	}
	want := []string{"configure", "doctor", "list"}
	slices.Sort(uses)
	if !slices.Equal(uses, want) {
		t.Fatalf("got %v want %v", uses, want)
	}
}
