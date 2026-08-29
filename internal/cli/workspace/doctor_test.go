package workspace

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

func TestDoctorNonInteractiveReportsAndFails(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "ra", Name: "active-sales", WorkspaceID: id, CurrentPath: filepath.Join(dir, "missing"),
	})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	err = doctorRun(cmd, "acme")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "1 issue(s) found") {
		t.Fatalf("err=%v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "path missing") || !strings.Contains(out, "active-sales") {
		t.Fatalf("out=%s", out)
	}
	s, _ = shared.Load()
	if _, err := shared.GetRepo(s, "ra"); err != nil {
		t.Fatal("non-interactive must not delete the registry entry")
	}
}

func TestDoctorDeclinedFixLeavesState(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "gone")
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "ra", Name: "gone", WorkspaceID: id, CurrentPath: missing,
	})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), &doctorPrompter{picks: []string{"ignore"}}))
	if err := doctorRun(cmd, "acme"); err != nil {
		t.Fatal(err)
	}
	s, _ = shared.Load()
	repo, err := shared.GetRepo(s, "ra")
	if err != nil {
		t.Fatal(err)
	}
	if repo.CurrentPath != missing {
		t.Fatalf("path=%q", repo.CurrentPath)
	}
}

func TestDoctorDropsUnknownLinkedRepo(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	ws, _ := shared.GetWorkspace(s, id)
	ws.LinkedRepos = []string{"ghost"}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), &doctorPrompter{confirm: true}))
	if err := doctorRun(cmd, "acme"); err != nil {
		t.Fatal(err)
	}
	s, _ = shared.Load()
	ws, _ = shared.GetWorkspace(s, id)
	if len(ws.LinkedRepos) != 0 {
		t.Fatalf("linked=%v", ws.LinkedRepos)
	}
}

func TestDoctorAddsMissingLinkedRepo(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	repoPath := filepath.Join(dir, "app")
	if err := os.MkdirAll(filepath.Join(repoPath, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(repoPath, ".git")
	m.Repo.LocalConfig["elegant-git.repo-id"] = "ra"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	s.Repositories["ra"] = &shared.Repository{Name: "a", WorkspaceID: id, CurrentPath: repoPath}
	ws, _ := shared.GetWorkspace(s, id)
	ws.LinkedRepos = nil
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), &doctorPrompter{confirm: true}))
	if err := doctorRun(cmd, "acme"); err != nil {
		t.Fatal(err)
	}
	s, _ = shared.Load()
	ws, _ = shared.GetWorkspace(s, id)
	if !containsID(ws.LinkedRepos, "ra") {
		t.Fatalf("linked=%v", ws.LinkedRepos)
	}
}

func TestDoctorNormalizesNamespaces(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
		Namespaces: []string{" github.com/acme ", "", "github.com/acme", "gitlab.com/acme"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), &doctorPrompter{confirm: true}))
	if err := doctorRun(cmd, "acme"); err != nil {
		t.Fatal(err)
	}
	s, _ = shared.Load()
	ws, _ := shared.GetWorkspace(s, id)
	if got := strings.Join(ws.Namespaces, ","); got != "github.com/acme,gitlab.com/acme" {
		t.Fatalf("namespaces=%v", ws.Namespaces)
	}
}

func TestDoctorRemovesMissingPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "ra", Name: "gone", WorkspaceID: id, CurrentPath: filepath.Join(dir, "gone"),
	})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), &doctorPrompter{
		picks: []string{"remove"},
	}))
	if err := doctorRun(cmd, "acme"); err != nil {
		t.Fatal(err)
	}
	s, _ = shared.Load()
	if _, err := shared.GetRepo(s, "ra"); err == nil {
		t.Fatal("expected registry entry removed")
	}
	ws, _ := shared.GetWorkspace(s, id)
	if containsID(ws.LinkedRepos, "ra") {
		t.Fatalf("still linked: %v", ws.LinkedRepos)
	}
}

func TestDoctorRelocatesMissingPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	newPath := filepath.Join(dir, "relocated")
	if err := os.MkdirAll(filepath.Join(newPath, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(newPath, ".git")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	oldPath := filepath.Join(dir, "old")
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "ra", Name: "app", WorkspaceID: id, CurrentPath: oldPath,
	})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), &doctorPrompter{
		picks:  []string{"path"},
		inputs: []string{newPath},
	}))
	if err := doctorRun(cmd, "acme"); err != nil {
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
	if m.Repo.LocalConfig["elegant-git.repo-id"] != "ra" {
		t.Fatalf("repo-id=%v", m.Repo.LocalConfig)
	}
}

func TestDoctorRelocateRejectsPathOwnedByOther(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	other := filepath.Join(dir, "other")
	if err := os.MkdirAll(filepath.Join(other, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(other, ".git")
	m.Repo.LocalConfig["elegant-git.repo-id"] = "rb"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	oldPath := filepath.Join(dir, "old")
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "ra", Name: "gone", WorkspaceID: id, CurrentPath: oldPath,
	})
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "rb", Name: "kept", WorkspaceID: id, CurrentPath: other,
	})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), &doctorPrompter{
		picks:  []string{"path"},
		inputs: []string{other},
	}))
	if err := doctorRun(cmd, "acme"); err != nil {
		t.Fatal(err)
	}
	s, _ = shared.Load()
	repo, err := shared.GetRepo(s, "ra")
	if err != nil {
		t.Fatal(err)
	}
	if repo.CurrentPath == other {
		t.Fatal("must not steal another repository's path")
	}
}

func TestDoctorRelocateRejectsForeignRepoID(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	newPath := filepath.Join(dir, "relocated")
	if err := os.MkdirAll(filepath.Join(newPath, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(newPath, ".git")
	m.Repo.LocalConfig["elegant-git.repo-id"] = "foreign"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	oldPath := filepath.Join(dir, "old")
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "ra", Name: "app", WorkspaceID: id, CurrentPath: oldPath,
	})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), &doctorPrompter{
		picks:  []string{"path"},
		inputs: []string{newPath},
	}))
	if err := doctorRun(cmd, "acme"); err != nil {
		t.Fatal(err)
	}
	s, _ = shared.Load()
	repo, err := shared.GetRepo(s, "ra")
	if err != nil {
		t.Fatal(err)
	}
	if repo.CurrentPath != oldPath {
		t.Fatalf("path=%q", repo.CurrentPath)
	}
	if m.Repo.LocalConfig["elegant-git.repo-id"] != "foreign" {
		t.Fatalf("repo-id overwritten: %v", m.Repo.LocalConfig)
	}
}

func TestDoctorIgnoresNotAGitWorkTree(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	notGit := filepath.Join(dir, "notgit")
	if err := os.MkdirAll(notGit, 0o755); err != nil {
		t.Fatal(err)
	}

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "ra", Name: "app", WorkspaceID: id, CurrentPath: notGit,
	})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), &doctorPrompter{picks: []string{"ignore"}}))
	if err := doctorRun(cmd, "acme"); err != nil {
		t.Fatal(err)
	}
	s, _ = shared.Load()
	if _, err := shared.GetRepo(s, "ra"); err != nil {
		t.Fatal("ignore must keep the registry entry")
	}
}

func TestDoctorOrphanClearsAndNormalizesNamespaces(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
		Namespaces: []string{" github.com/acme ", "github.com/acme"},
	})
	if err != nil {
		t.Fatal(err)
	}
	s.Repositories["orphan"] = &shared.Repository{
		Name: "x", WorkspaceID: "ghost", CurrentPath: dir,
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), &doctorPrompter{
		confirm: true,
		picks:   []string{"clear"},
	}))
	if err := doctorRun(cmd, "acme"); err != nil {
		t.Fatal(err)
	}
	s, _ = shared.Load()
	repo, err := shared.GetRepo(s, "orphan")
	if err != nil {
		t.Fatal(err)
	}
	if repo.WorkspaceID != "" {
		t.Fatalf("workspace_id=%q", repo.WorkspaceID)
	}
	ws, _ := shared.GetWorkspace(s, id)
	if got := strings.Join(ws.Namespaces, ","); got != "github.com/acme" {
		t.Fatalf("namespaces=%v", ws.Namespaces)
	}
}

func TestDoctorStampsRepoID(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	repoPath := filepath.Join(dir, "app")
	if err := os.MkdirAll(filepath.Join(repoPath, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(repoPath, ".git")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "ra", Name: "app", WorkspaceID: id, CurrentPath: repoPath,
	})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), &doctorPrompter{confirm: true}))
	if err := doctorRun(cmd, "acme"); err != nil {
		t.Fatal(err)
	}
	if m.Repo.LocalConfig["elegant-git.repo-id"] != "ra" {
		t.Fatalf("config=%v", m.Repo.LocalConfig)
	}
}

func TestDoctorRewritesPerRepoWorkspaceID(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	repoPath := filepath.Join(dir, "app")
	gitDir := filepath.Join(repoPath, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(dir, "repo-state.json"))

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	m.Repo.LocalConfig["elegant-git.repo-id"] = "ra"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	if err := memrepo.Save(gitDir, &memrepo.State{
		SchemaVersion: memrepo.SchemaVersion,
		RepoID:        "ra",
		WorkspaceID:   "stale",
	}); err != nil {
		t.Fatal(err)
	}

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "ra", Name: "app", WorkspaceID: id, CurrentPath: repoPath,
	})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), &doctorPrompter{confirm: true}))
	if err := doctorRun(cmd, "acme"); err != nil {
		t.Fatal(err)
	}
	per, err := memrepo.Load(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	if per.WorkspaceID != id {
		t.Fatalf("workspace_id=%q want %q", per.WorkspaceID, id)
	}
}

func TestDoctorHealthy(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	_, err = shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := doctorRun(cmd, "acme"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "looks healthy") {
		t.Fatalf("out=%s", buf.String())
	}
}

func TestDoctorRequiresNameNonInteractive(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	_, err = shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	err = doctorRun(cmd, "")
	if err == nil || err.Error() != "workspace name is required" {
		t.Fatalf("got %v", err)
	}
}

func containsID(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

type doctorPrompter struct {
	confirm  bool
	picks    []string
	pickIdx  int
	optional []string
	optIdx   int
	edits    []string
	editIdx  int
	inputs   []string
	inputIdx int
}

func (p *doctorPrompter) String(string, string) (string, error) {
	if p.inputIdx < len(p.inputs) {
		v := p.inputs[p.inputIdx]
		p.inputIdx++
		return v, nil
	}
	return "", io.EOF
}
func (p *doctorPrompter) Confirm(string, bool) (bool, error) { return p.confirm, nil }
func (p *doctorPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}
func (p *doctorPrompter) Pick(_ string, choices []prompt.Choice, _ string) (string, error) {
	if p.pickIdx < len(p.picks) {
		v := p.picks[p.pickIdx]
		p.pickIdx++
		return v, nil
	}
	if len(choices) > 0 {
		return choices[0].Value, nil
	}
	return "", prompt.ErrNonInteractive
}
func (p *doctorPrompter) Required(string, string) error { return nil }
func (p *doctorPrompter) EditOrAccept(_ string, suggested string) (string, error) {
	if p.editIdx < len(p.edits) {
		v := p.edits[p.editIdx]
		p.editIdx++
		return v, nil
	}
	return suggested, nil
}
func (p *doctorPrompter) Optional(string, string) (string, error) {
	if p.optIdx < len(p.optional) {
		v := p.optional[p.optIdx]
		p.optIdx++
		return v, nil
	}
	return "", nil
}
func (p *doctorPrompter) Closed(string, []string, string, bool) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *doctorPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}
