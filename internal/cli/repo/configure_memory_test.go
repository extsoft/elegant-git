package repo

import (
	"path/filepath"
	"testing"

	"github.com/bees-hive/elegant-git/internal/cli/sources"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/prompt"
)

func TestResolveProfileCreateNew(t *testing.T) {
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
	id, prof, err := resolveProfile(s, sources.ProfileCreateNew, p)
	if err != nil {
		t.Fatal(err)
	}
	if prof.Name != "work" || id == "" {
		t.Fatalf("id=%q prof=%+v", id, prof)
	}
}

type configureRecordingPrompter struct {
	editValues []string
	editIdx    int
}

func (p *configureRecordingPrompter) String(q, def string) (string, error) {
	return def, nil
}
func (p *configureRecordingPrompter) Confirm(string, bool) (bool, error) { return false, nil }
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

func TestResolveProfileCreateNewNonInteractive(t *testing.T) {
	s, _ := shared.Load()
	_, _, err := resolveProfile(s, sources.ProfileCreateNew, prompt.NewNonInteractive())
	if err == nil {
		t.Fatal("expected error")
	}
}
