package workspace

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/prompt"
)

func TestDeleteSpecKeepsCLIArg(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, _ := shared.Load()
	if _, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "dz", UserName: "D", UserEmail: "d@x.com",
	}); err != nil {
		t.Fatal(err)
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	var name string
	spec := deleteSpec(&name)
	p := &deleteRecordingPrompter{}
	if err := argspec.Resolve(context.Background(), p, []string{"dz"}, spec); err != nil {
		t.Fatal(err)
	}
	if name != "dz" {
		t.Fatalf("name=%q", name)
	}
	if p.pickIdx != 0 {
		t.Fatal("unexpected picker")
	}
}

func TestDeleteSpecPicksWhenMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, _ := shared.Load()
	if _, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "dz", UserName: "D", UserEmail: "d@x.com",
	}); err != nil {
		t.Fatal(err)
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	var name string
	spec := deleteSpec(&name)
	p := &deleteRecordingPrompter{pickValues: []string{"dz"}}
	if err := argspec.Resolve(context.Background(), p, nil, spec); err != nil {
		t.Fatal(err)
	}
	if name != "dz" {
		t.Fatalf("name=%q", name)
	}
}

func TestDeleteHelp(t *testing.T) {
	c := newDeleteCommand()
	var buf bytes.Buffer
	c.SetOut(&buf)
	c.SetArgs([]string{"--help"})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"Usage:", "delete <name>", "Deletes a workspace", "--yes"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestDeleteSummaryConfirmYesSkipsPrompt(t *testing.T) {
	s := emptyShared(t)
	id, ws := makeLinked(t, s)
	p := &deleteRecordingPrompter{confirm: false}
	var buf bytes.Buffer
	cmd := newDeleteCommand()
	cmd.SetOut(&buf)
	if err := deleteSummaryConfirm(cmd, s, id, ws, true, p); err != nil {
		t.Fatal(err)
	}
	if p.confirmCalls != 0 {
		t.Fatalf("confirm calls=%d", p.confirmCalls)
	}
	if !strings.Contains(buf.String(), `Workspace "acme" will be deleted`) {
		t.Fatalf("summary missing: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "Unlinks 1 repository") {
		t.Fatalf("unlink block missing: %q", buf.String())
	}
}

func TestDeleteSummaryConfirmNonInteractiveRequiresYes(t *testing.T) {
	s := emptyShared(t)
	id, ws := makeLinked(t, s)
	var buf bytes.Buffer
	cmd := newDeleteCommand()
	cmd.SetOut(&buf)
	err := deleteSummaryConfirm(cmd, s, id, ws, false, prompt.NewNonInteractive())
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("got %v", err)
	}
}

func TestDeleteSummaryConfirmDeclineAborts(t *testing.T) {
	s := emptyShared(t)
	id, ws := makeLinked(t, s)
	p := &deleteRecordingPrompter{confirm: false}
	cmd := newDeleteCommand()
	cmd.SetOut(&bytes.Buffer{})
	err := deleteSummaryConfirm(cmd, s, id, ws, false, p)
	if err == nil {
		t.Fatal("expected abort")
	}
}

func emptyShared(t *testing.T) *shared.State {
	t.Helper()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(t.TempDir(), "state.json"))
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func makeLinked(t *testing.T, s *shared.State) (string, *shared.Workspace) {
	t.Helper()
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "Ann Acme", UserEmail: "ann@acme.io",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "r1", Name: "app", WorkspaceID: id, CurrentPath: "/Users/ann/src/app",
	}); err != nil {
		t.Fatal(err)
	}
	return id, s.Workspaces[id]
}

type deleteRecordingPrompter struct {
	pickValues   []string
	pickIdx      int
	confirm      bool
	confirmCalls int
}

func (p *deleteRecordingPrompter) String(string, string) (string, error) { return "", nil }
func (p *deleteRecordingPrompter) Confirm(string, bool) (bool, error) {
	p.confirmCalls++
	return p.confirm, nil
}
func (p *deleteRecordingPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}
func (p *deleteRecordingPrompter) Required(string, string) error { return nil }
func (p *deleteRecordingPrompter) Optional(string, string) (string, error) {
	return "", nil
}
func (p *deleteRecordingPrompter) Closed(string, []string, string, bool) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *deleteRecordingPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}
func (p *deleteRecordingPrompter) EditOrAccept(string, string) (string, error) {
	return "", nil
}
func (p *deleteRecordingPrompter) Pick(string, []prompt.Choice, string) (string, error) {
	if p.pickIdx < len(p.pickValues) {
		v := p.pickValues[p.pickIdx]
		p.pickIdx++
		return v, nil
	}
	return "", nil
}
