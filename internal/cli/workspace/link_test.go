package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

func TestLinkRunRequiresGit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	t.Chdir(dir)
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = ""
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	cmd := &cobra.Command{}
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	err := linkRun(cmd, "acme")
	if err == nil || err.Error() != "not a git repository" {
		t.Fatalf("got %v", err)
	}
}

func TestLinkRunAppliesWorkspace(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	repo := filepath.Join(dir, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(repo, ".git")
	m.Outputs["rev-parse --show-toplevel"] = repo
	m.Repo.LocalConfig["elegant-git.repo-id"] = "ra"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com", SigningKey: "KEY",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(prompt.WithPrompter(context.Background(), &linkPrompter{confirm: true}))
	if err := linkRun(cmd, "acme"); err != nil {
		t.Fatal(err)
	}
	s, err = shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	reg, err := shared.GetRepo(s, "ra")
	if err != nil {
		t.Fatal(err)
	}
	if reg.WorkspaceID != id {
		t.Fatalf("workspace_id=%q want %q", reg.WorkspaceID, id)
	}
	if m.Repo.LocalConfig["user.name"] != "U" || m.Repo.LocalConfig["user.email"] != "u@e.com" {
		t.Fatalf("config=%v", m.Repo.LocalConfig)
	}
}

func TestLinkRunOverrideRejected(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	repo := filepath.Join(dir, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(repo, ".git")
	m.Outputs["rev-parse --show-toplevel"] = repo
	m.Repo.LocalConfig["elegant-git.repo-id"] = "ra"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	oldID, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "old", UserName: "O", UserEmail: "o@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "new", UserName: "N", UserEmail: "n@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{ID: "ra", Name: "repo", WorkspaceID: oldID, CurrentPath: repo})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(prompt.WithPrompter(context.Background(), &linkPrompter{confirm: false}))
	if err := linkRun(cmd, "new"); err != nil {
		t.Fatal(err)
	}
	s, err = shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	reg, _ := shared.GetRepo(s, "ra")
	if reg.WorkspaceID != oldID {
		t.Fatalf("workspace_id=%q want old %q", reg.WorkspaceID, oldID)
	}
}

type linkPrompter struct {
	confirm bool
}

func (p *linkPrompter) String(string, string) (string, error) { return "", nil }
func (p *linkPrompter) Confirm(string, bool) (bool, error)    { return p.confirm, nil }
func (p *linkPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}
func (p *linkPrompter) Pick(string, []prompt.Choice, string) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *linkPrompter) Required(string, string) error               { return nil }
func (p *linkPrompter) EditOrAccept(string, string) (string, error) { return "", nil }
func (p *linkPrompter) Optional(string, string) (string, error)     { return "", nil }
func (p *linkPrompter) Closed(string, []string, string, bool) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *linkPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}
