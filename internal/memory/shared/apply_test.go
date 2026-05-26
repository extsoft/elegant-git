package shared

import (
	"testing"

	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/prompt"
)

type fakePrompter struct {
	confirmYes bool
	editVal    string
}

func (f fakePrompter) String(string, string) (string, error) {
	return f.editVal, prompt.ErrNonInteractive
}
func (f fakePrompter) Confirm(string) (bool, error)         { return f.confirmYes, nil }
func (f fakePrompter) Choose(string, []string) (int, error) { return 0, prompt.ErrNonInteractive }
func (f fakePrompter) Required(string, string) error        { return prompt.ErrNonInteractive }
func (f fakePrompter) EditOrAccept(string, suggested string) (string, error) {
	if f.editVal != "" {
		return f.editVal, nil
	}
	return "", nil
}
func (f fakePrompter) BatchChoice(string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}

func TestApplyProfileRequiredKeys(t *testing.T) {
	m := git.NewMemoryRunner()
	git.Use(m)
	prof := &Profile{UserName: "Jane", UserEmail: "jane@example.com", SigningKey: "ABC"}
	if err := ApplyProfile(prof, fakePrompter{}, nil); err != nil {
		t.Fatal(err)
	}
	if m.Repo.LocalConfig["user.name"] != "Jane" {
		t.Fatalf("user.name = %q", m.Repo.LocalConfig["user.name"])
	}
	if _, ok := m.Repo.LocalConfig["user.signingkey"]; ok {
		t.Fatal("signing key should not be applied without value confirmation path")
	}
}

func TestApplyProfileAutoSkipWhenEqual(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Repo.LocalConfig["user.name"] = "Jane"
	m.Repo.LocalConfig["user.email"] = "jane@example.com"
	git.Use(m)
	prof := &Profile{UserName: "Jane", UserEmail: "jane@example.com"}
	before := len(m.Calls)
	if err := ApplyProfile(prof, fakePrompter{}, nil); err != nil {
		t.Fatal(err)
	}
	for _, c := range m.Calls[before:] {
		if len(c.Args) >= 2 && c.Args[0] == "config" && c.Args[1] == "user.name" {
			t.Fatal("should not write user.name when equal")
		}
	}
}

func TestApplyProfileForceWritesOptional(t *testing.T) {
	m := git.NewMemoryRunner()
	git.Use(m)
	prof := &Profile{UserName: "J", UserEmail: "j@e.com", SigningKey: "KEY"}
	if err := ApplyProfile(prof, fakePrompter{}, &Apply{Force: true}); err != nil {
		t.Fatal(err)
	}
	if m.Repo.LocalConfig["user.signingkey"] != "KEY" {
		t.Fatalf("signingkey = %q", m.Repo.LocalConfig["user.signingkey"])
	}
}

func TestApplyProfileSkipHalts(t *testing.T) {
	m := git.NewMemoryRunner()
	git.Use(m)
	prof := &Profile{UserName: "Jane", UserEmail: "jane@example.com"}
	if err := ApplyProfile(prof, fakePrompter{}, &Apply{Skip: true}); err != nil {
		t.Fatal(err)
	}
	if len(m.Repo.LocalConfig) != 0 {
		t.Fatalf("expected no writes, got %v", m.Repo.LocalConfig)
	}
}
