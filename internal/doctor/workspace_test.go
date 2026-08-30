package doctor

import (
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
)

func TestApplyWorkspaceNameRejectsReserved(t *testing.T) {
	ws := &shared.Workspace{Name: "old"}
	for _, name := range []string{shared.SelectorAll, shared.SelectorCurrent, ""} {
		err := applyWorkspaceName(ws, name)
		if err == nil {
			t.Fatalf("name %q: expected error", name)
		}
		if ws.Name != "old" {
			t.Fatalf("name %q mutated workspace to %q", name, ws.Name)
		}
	}
}

func TestApplyWorkspaceNameAcceptsValid(t *testing.T) {
	ws := &shared.Workspace{Name: "old"}
	if err := applyWorkspaceName(ws, "  work  "); err != nil {
		t.Fatal(err)
	}
	if ws.Name != "work" {
		t.Fatalf("got %q", ws.Name)
	}
}

func TestWorkspaceDuplicateRenameRejectsReserved(t *testing.T) {
	s := &shared.State{
		Workspaces: map[string]*shared.Workspace{
			"a": {Name: "dup", UserName: "A", UserEmail: "a@x.com"},
			"b": {Name: "dup", UserName: "B", UserEmail: "b@x.com"},
		},
		Repositories: map[string]*shared.Repository{},
	}
	p := &reservedNamePrompter{name: shared.SelectorAll}
	findings := Workspace(s, "a", p)
	var apply func() error
	for _, f := range findings {
		if strings.Contains(f.Problem, "also used") {
			apply = f.Apply
			break
		}
	}
	if apply == nil {
		t.Fatal("missing duplicate-name finding")
	}
	err := apply()
	if err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("got %v", err)
	}
}

type reservedNamePrompter struct {
	name string
}

func (p *reservedNamePrompter) String(string, string) (string, error) { return "", nil }
func (p *reservedNamePrompter) Confirm(string, bool) (bool, error)    { return false, nil }
func (p *reservedNamePrompter) Choose(string, []string) (int, error) {
	return -1, nil
}
func (p *reservedNamePrompter) Pick(string, []prompt.Choice, string) (string, error) {
	return "", nil
}
func (p *reservedNamePrompter) Required(string, string) error { return nil }
func (p *reservedNamePrompter) EditOrAccept(string, string) (string, error) {
	return p.name, nil
}
func (p *reservedNamePrompter) Optional(string, string) (string, error) { return "", nil }
func (p *reservedNamePrompter) Closed(string, []string, string, bool) (string, error) {
	return "", nil
}
func (p *reservedNamePrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}
