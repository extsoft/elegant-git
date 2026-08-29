package memory

import (
	"testing"

	workspacecmd "github.com/extsoft/elegant-git/internal/cli/workspace"
	"github.com/extsoft/elegant-git/internal/memory/shared"
)

func TestMemoryWorkspacesDelegatesToWorkspacePackage(t *testing.T) {
	s := &shared.State{
		Workspaces: map[string]*shared.Workspace{
			"id-a": {Name: "alpha", UserName: "A", UserEmail: "a@x.com", LinkedRepos: []string{}},
		},
	}
	// Smoke: workspace package renderers are the implementation behind memory workspaces.
	if err := workspacecmd.PrintList(ioDiscard{}, s, "table"); err != nil {
		t.Fatal(err)
	}
	if err := workspacecmd.PrintDetails(ioDiscard{}, s, "alpha", "table"); err != nil {
		t.Fatal(err)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
