package repo

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

func TestDoctorUpdatesStalePath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	oldPath := filepath.Join(dir, "old")
	newPath := filepath.Join(dir, "new")
	if err := os.MkdirAll(filepath.Join(newPath, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(newPath)

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(newPath, ".git")
	m.Repo.LocalConfig["elegant-git.repo-id"] = "ra"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	wsID, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "ra", Name: "app", WorkspaceID: wsID, CurrentPath: oldPath,
	}); err != nil {
		t.Fatal(err)
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), &repoDoctorPrompter{confirm: true}))
	if err := doctorRun(cmd); err != nil {
		t.Fatal(err)
	}
	s, _ = shared.Load()
	repo, err := shared.GetRepo(s, "ra")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(newPath)
	if repo.CurrentPath != want {
		t.Fatalf("path=%q want %q", repo.CurrentPath, want)
	}
	if len(repo.PathHistory) == 0 || repo.PathHistory[0] != oldPath {
		t.Fatalf("history=%v", repo.PathHistory)
	}
}

func TestDoctorStampsRepoIDFromPathMatch(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(dir, ".git")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	wsID, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(dir)
	if err := shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "ra", Name: "app", WorkspaceID: wsID, CurrentPath: abs,
	}); err != nil {
		t.Fatal(err)
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), &repoDoctorPrompter{confirm: true}))
	if err := doctorRun(cmd); err != nil {
		t.Fatal(err)
	}
	if m.Repo.LocalConfig["elegant-git.repo-id"] != "ra" {
		t.Fatalf("config=%v", m.Repo.LocalConfig)
	}
}

func TestDoctorNonInteractiveReportsPathDrift(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	oldPath := filepath.Join(dir, "old")
	newPath := filepath.Join(dir, "new")
	if err := os.MkdirAll(filepath.Join(newPath, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(newPath)

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(newPath, ".git")
	m.Repo.LocalConfig["elegant-git.repo-id"] = "ra"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	wsID, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "ra", Name: "app", WorkspaceID: wsID, CurrentPath: oldPath,
	})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	err = doctorRun(cmd)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "issue(s) found") {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(buf.String(), "differs from cwd") {
		t.Fatalf("out=%s", buf.String())
	}
	s, _ = shared.Load()
	repo, _ := shared.GetRepo(s, "ra")
	if repo.CurrentPath != oldPath {
		t.Fatal("non-interactive must not change path")
	}
}

func TestDoctorHealthy(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(dir, ".git")
	m.Repo.LocalConfig["elegant-git.repo-id"] = "ra"
	m.Repo.LocalConfig["user.name"] = "U"
	m.Repo.LocalConfig["user.email"] = "u@e.com"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	wsID, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(dir)
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "ra", Name: "app", WorkspaceID: wsID, CurrentPath: abs,
	})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := doctorRun(cmd); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "looks healthy") {
		t.Fatalf("out=%s", buf.String())
	}
}

type repoDoctorPrompter struct {
	confirm bool
}

func (p *repoDoctorPrompter) String(string, string) (string, error) { return "", nil }
func (p *repoDoctorPrompter) Confirm(string, bool) (bool, error)    { return p.confirm, nil }
func (p *repoDoctorPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}
func (p *repoDoctorPrompter) Pick(string, []prompt.Choice, string) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *repoDoctorPrompter) Required(string, string) error               { return nil }
func (p *repoDoctorPrompter) EditOrAccept(string, string) (string, error) { return "", nil }
func (p *repoDoctorPrompter) Optional(string, string) (string, error)     { return "", nil }
func (p *repoDoctorPrompter) Closed(string, []string, string, bool) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *repoDoctorPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}
