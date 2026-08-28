package repo

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bees-hive/elegant-git/internal/cli/sources"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

func TestResolveWorkspaceCreateNew(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	m := git.NewMemoryRunner()
	m.GlobalConfig["user.name"] = "User"
	m.GlobalConfig["user.email"] = "user@example.com"
	git.Use(m)

	s, _ := shared.Load()

	p := &configureRecordingPrompter{
		editValues: []string{"work", "", "", "", "", ""},
	}
	id, prof, err := resolveWorkspace(s, sources.WorkspaceCreateNew, "https://github.com/acme/app.git", p)
	if err != nil {
		t.Fatal(err)
	}
	if prof.Name != "work" || id == "" {
		t.Fatalf("id=%q prof=%+v", id, prof)
	}
	if len(prof.Namespaces) != 1 || prof.Namespaces[0] != "github.com/acme" {
		t.Fatalf("namespaces=%v", prof.Namespaces)
	}
}

func TestResolveWorkspaceFromNamespaceUnique(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s := sharedEmpty(t)
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com", Namespaces: []string{"github.com/acme"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	s, _ = shared.Load()

	gotID, ws, err := resolveWorkspace(s, "", "https://github.com/acme/app.git", prompt.NewNonInteractive())
	if err != nil {
		t.Fatal(err)
	}
	if gotID != id || ws.Name != "acme" {
		t.Fatalf("got id=%q ws=%+v", gotID, ws)
	}
}

func TestResolveWorkspaceFromNamespaceMissingNonInteractive(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s := sharedEmpty(t)
	id, ws, err := resolveWorkspace(s, "", "https://github.com/acme/app.git", prompt.NewNonInteractive())
	if err != nil {
		t.Fatal(err)
	}
	if id != "" || ws != nil {
		t.Fatalf("want no workspace assigned; got id=%q ws=%v", id, ws)
	}
}

func TestConfigureWithMemoryNoWorkspace(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	git.Use(git.NewMemoryRunner())
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	cmd := &cobra.Command{}
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	assigned, err := configureWithMemory(cmd, "")
	if err != nil {
		t.Fatal(err)
	}
	if assigned {
		t.Fatal("expected no workspace assigned")
	}
}

func TestDisambiguateCloneArgs(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s := sharedEmpty(t)
	_, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{Name: "work", UserName: "U", UserEmail: "u@e.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	ws, directory := "mydir", ""
	disambiguateCloneArgs(&ws, &directory)
	if ws != "" || directory != "mydir" {
		t.Fatalf("want directory=mydir workspace empty; got workspace=%q directory=%q", ws, directory)
	}

	ws, directory = "work", ""
	disambiguateCloneArgs(&ws, &directory)
	if ws != "work" || directory != "" {
		t.Fatalf("want workspace=work; got workspace=%q directory=%q", ws, directory)
	}

	ws, directory = "work", "other"
	disambiguateCloneArgs(&ws, &directory)
	if ws != "work" || directory != "other" {
		t.Fatalf("both set: workspace=%q directory=%q", ws, directory)
	}
}

func sharedEmpty(t *testing.T) *shared.State {
	t.Helper()
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

type configureRecordingPrompter struct {
	editValues   []string
	editIdx      int
	confirm      bool
	confirmCalls int
}

func (p *configureRecordingPrompter) String(q, def string) (string, error) {
	return def, nil
}
func (p *configureRecordingPrompter) Confirm(string, bool) (bool, error) {
	p.confirmCalls++
	return p.confirm, nil
}
func (p *configureRecordingPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}
func (p *configureRecordingPrompter) Pick(string, []prompt.Choice, string) (string, error) {
	return "", nil
}
func (p *configureRecordingPrompter) Required(string, string) error { return nil }
func (p *configureRecordingPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}
func (p *configureRecordingPrompter) EditOrAccept(label, suggested string) (string, error) {
	if p.editIdx < len(p.editValues) {
		v := p.editValues[p.editIdx]
		p.editIdx++
		if v != "" {
			return v, nil
		}
	}
	return suggested, nil
}

func (p *configureRecordingPrompter) Optional(string, suggested string) (string, error) {
	if p.editIdx < len(p.editValues) {
		v := p.editValues[p.editIdx]
		p.editIdx++
		return v, nil
	}
	return "", nil
}

func (p *configureRecordingPrompter) Closed(string, []string, string, bool) (string, error) {
	return "", prompt.ErrNonInteractive
}

func TestResolveWorkspaceCreateNewNonInteractive(t *testing.T) {
	s, _ := shared.Load()
	_, _, err := resolveWorkspace(s, sources.WorkspaceCreateNew, "", prompt.NewNonInteractive())
	if err == nil {
		t.Fatal("expected error")
	}
}
