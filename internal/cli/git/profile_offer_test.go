package git

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

func TestOfferCreateProfileFromGlobalSkipsDuplicate(t *testing.T) {
	t.Setenv("ELEGANT_GIT_STATE_FILE", t.TempDir()+"/state.json")
	m := git.NewMemoryRunner()
	m.GlobalConfig["user.name"] = "Jane"
	m.GlobalConfig["user.email"] = "jane@example.com"
	git.Use(m)
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	_, err = shared.CreateProfile(s, shared.CreateProfileInput{
		Name: "jane", UserName: "Jane", UserEmail: "jane@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := offerCreateProfileFromGlobal(cmd); err != nil {
		t.Fatal(err)
	}
	if len(s.Profiles) != 1 {
		t.Fatalf("profiles = %d", len(s.Profiles))
	}
}

func TestOfferCreateProfileFromGlobalCreates(t *testing.T) {
	t.Setenv("ELEGANT_GIT_STATE_FILE", t.TempDir()+"/state.json")
	m := git.NewMemoryRunner()
	m.GlobalConfig["user.name"] = "Bob"
	m.GlobalConfig["user.email"] = "bob@example.com"
	git.Use(m)
	in := strings.NewReader("y\nbob\n")
	p := prompt.NewTTY(in, &bytes.Buffer{})
	cmd := &cobra.Command{}
	cmd.SetContext(prompt.WithPrompter(context.Background(), p))
	if err := offerCreateProfileFromGlobal(cmd); err != nil {
		t.Fatal(err)
	}
	s, _ := shared.Load()
	if len(s.Profiles) != 1 {
		t.Fatalf("profiles = %d", len(s.Profiles))
	}
}
