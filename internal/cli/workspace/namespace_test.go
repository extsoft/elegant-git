package workspace

import (
	"path/filepath"
	"testing"

	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
)

func TestCaptureNamespaceConfirm(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{Name: "acme", UserName: "U", UserEmail: "u@e.com"})
	if err != nil {
		t.Fatal(err)
	}
	ws := s.Workspaces[id]
	p := &namespacePrompter{confirm: true}
	if err := CaptureNamespace(s, id, ws, "git@github.com:acme/app.git", p); err != nil {
		t.Fatal(err)
	}
	if len(ws.Namespaces) != 1 || ws.Namespaces[0] != "github.com/acme" {
		t.Fatalf("namespaces=%v", ws.Namespaces)
	}
}

func TestCaptureNamespaceSkipWhenPresent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com", Namespaces: []string{"github.com/acme"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ws := s.Workspaces[id]
	p := &namespacePrompter{confirm: false}
	if err := CaptureNamespace(s, id, ws, "https://github.com/acme/app.git", p); err != nil {
		t.Fatal(err)
	}
	if p.confirmCalls != 0 {
		t.Fatalf("unexpected confirm calls: %d", p.confirmCalls)
	}
}

func TestCaptureNamespaceNonInteractiveRecords(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{Name: "acme", UserName: "U", UserEmail: "u@e.com"})
	if err != nil {
		t.Fatal(err)
	}
	ws := s.Workspaces[id]
	if err := CaptureNamespace(s, id, ws, "git@github.com:extsoft/elegant-git.git", prompt.NewNonInteractive()); err != nil {
		t.Fatal(err)
	}
	if len(ws.Namespaces) != 1 || ws.Namespaces[0] != "github.com/extsoft" {
		t.Fatalf("namespaces=%v", ws.Namespaces)
	}
}

type namespacePrompter struct {
	confirm      bool
	confirmCalls int
}

func (p *namespacePrompter) String(string, string) (string, error) { return "", nil }
func (p *namespacePrompter) Confirm(string, bool) (bool, error) {
	p.confirmCalls++
	return p.confirm, nil
}
func (p *namespacePrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}
func (p *namespacePrompter) Pick(string, []prompt.Choice, string) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *namespacePrompter) Required(string, string) error { return nil }
func (p *namespacePrompter) EditOrAccept(string, string) (string, error) {
	return "", nil
}
func (p *namespacePrompter) Optional(string, string) (string, error) { return "", nil }
func (p *namespacePrompter) Closed(string, []string, string, bool) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *namespacePrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}
