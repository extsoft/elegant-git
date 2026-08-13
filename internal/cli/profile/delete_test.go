package profile

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
	if _, err := shared.CreateProfile(s, shared.CreateProfileInput{
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
	if _, err := shared.CreateProfile(s, shared.CreateProfileInput{
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
	for _, want := range []string{"Usage:", "delete [name]", "Deletes a profile", "no repositories are linked"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

type deleteRecordingPrompter struct {
	pickValues []string
	pickIdx    int
}

func (p *deleteRecordingPrompter) String(string, string) (string, error) { return "", nil }
func (p *deleteRecordingPrompter) Confirm(string, bool) (bool, error)    { return false, nil }
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
func (p *deleteRecordingPrompter) Pick(string, []prompt.Choice) (string, error) {
	if p.pickIdx < len(p.pickValues) {
		v := p.pickValues[p.pickIdx]
		p.pickIdx++
		return v, nil
	}
	return "", nil
}
