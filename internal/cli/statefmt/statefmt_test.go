package statefmt

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/memory/shared"
)

func TestReposWithWorkspace(t *testing.T) {
	s := &shared.State{
		Repositories: map[string]*shared.Repository{
			"a": {WorkspaceID: "p"},
			"b": {WorkspaceID: ""},
			"c": nil,
		},
	}
	if n := ReposWithWorkspace(s); n != 1 {
		t.Fatalf("got %d want 1", n)
	}
}

func TestFileStatusLine(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "nope.json")
	if !strings.Contains(FileStatusLine(missing), "not created yet") {
		t.Fatal(missing)
	}
	exists := filepath.Join(dir, "exists.json")
	if err := os.WriteFile(exists, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if FileStatusLine(exists) != exists {
		t.Fatalf("got %q", FileStatusLine(exists))
	}
}

func TestPrintBlockAndCatalog(t *testing.T) {
	var buf bytes.Buffer
	PrintCatalog(&buf, "", []Item{
		{Heading: "alpha", Fields: []Field{
			{Key: "identity", Value: "A <a@x.com>"},
			{Key: "repositories", Value: "0"},
			{Key: "explore", Value: "eg workspace list alpha"},
		}},
		{Heading: "beta", Fields: []Field{
			{Key: "identity", Value: "B <b@x.com>"},
			{Key: "repositories", Value: "1"},
			{Key: "explore", Value: "eg workspace list beta"},
		}},
	})
	want := strings.Join([]string{
		"alpha",
		"  identity:     A <a@x.com>",
		"  repositories: 0",
		"  explore:      eg workspace list alpha",
		"",
		"beta",
		"  identity:     B <b@x.com>",
		"  repositories: 1",
		"  explore:      eg workspace list beta",
		"",
	}, "\n")
	if buf.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestPrintFurtherSteps(t *testing.T) {
	var buf bytes.Buffer
	PrintFurtherSteps(&buf, nil)
	if buf.Len() != 0 {
		t.Fatalf("empty steps: %q", buf.String())
	}
	PrintFurtherSteps(&buf, []Step{
		{Command: "eg workspace list all", Comment: "the identities you commit with"},
		{Command: "eg repo list all", Comment: "the repositories Elegant Git looks after"},
	})
	want := strings.Join([]string{
		"",
		"Further steps:",
		"  eg workspace list all  the identities you commit with",
		"  eg repo list all       the repositories Elegant Git looks after",
		"",
	}, "\n")
	if buf.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestPrintLines(t *testing.T) {
	var buf bytes.Buffer
	PrintLines(&buf, " M first.go\n M second.go\n")
	if buf.String() != " M first.go\n M second.go\n" {
		t.Fatalf("got %q", buf.String())
	}
}

func TestWorkspaceName(t *testing.T) {
	s := &shared.State{
		Workspaces: map[string]*shared.Workspace{
			"p1": {Name: "work"},
		},
	}
	if got := WorkspaceName(s, ""); got != "(none)" {
		t.Fatalf("empty = %q", got)
	}
	if got := WorkspaceName(s, "p1"); got != "work" {
		t.Fatalf("known = %q", got)
	}
	if got := WorkspaceName(s, "missing"); got != "" {
		t.Fatalf("missing = %q", got)
	}
}
