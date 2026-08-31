package self

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

func TestOfferCreateWorkspaceFromGlobalSkipsDuplicate(t *testing.T) {
	t.Setenv("ELEGANT_GIT_STATE_FILE", t.TempDir()+"/state.json")
	m := git.NewMemoryRunner()
	m.GlobalConfig["user.name"] = "Jane"
	m.GlobalConfig["user.email"] = "jane@example.com"
	git.Use(m)
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	_, err = shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
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
	if err := offerCreateWorkspaceFromGlobal(cmd); err != nil {
		t.Fatal(err)
	}
	if len(s.Workspaces) != 1 {
		t.Fatalf("workspaces = %d", len(s.Workspaces))
	}
}

func TestOfferCreateWorkspaceFromGlobalCreates(t *testing.T) {
	t.Setenv("ELEGANT_GIT_STATE_FILE", t.TempDir()+"/state.json")
	m := git.NewMemoryRunner()
	m.GlobalConfig["user.name"] = "Bob"
	m.GlobalConfig["user.email"] = "bob@example.com"
	git.Use(m)
	in := strings.NewReader("y\nbob\n")
	p := prompt.NewTTY(in, &bytes.Buffer{})
	cmd := &cobra.Command{}
	cmd.SetContext(prompt.WithPrompter(context.Background(), p))
	if err := offerCreateWorkspaceFromGlobal(cmd); err != nil {
		t.Fatal(err)
	}
	s, _ := shared.Load()
	if len(s.Workspaces) != 1 {
		t.Fatalf("workspaces = %d", len(s.Workspaces))
	}
}
