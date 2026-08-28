package memory

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bees-hive/elegant-git/internal/memory/shared"
)

func TestListWorkspacesTable(t *testing.T) {
	s := &shared.State{
		Workspaces: map[string]*shared.Workspace{
			"id-b": {Name: "beta", UserName: "B", UserEmail: "b@x.com", LinkedRepos: []string{"r1"}},
			"id-a": {Name: "alpha", UserName: "A", UserEmail: "a@x.com", LinkedRepos: []string{}},
		},
	}
	var buf bytes.Buffer
	if err := listWorkspaces(&buf, s, "table"); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines: %q", lines)
	}
	if lines[0] != "alpha\tA <a@x.com>\t0 repo(s)" {
		t.Fatalf("first line: %q", lines[0])
	}
	if lines[1] != "beta\tB <b@x.com>\t1 repo(s)" {
		t.Fatalf("second line: %q", lines[1])
	}
}

func TestListWorkspacesJSON(t *testing.T) {
	s := &shared.State{
		Workspaces: map[string]*shared.Workspace{
			"id-a": {Name: "alpha", UserName: "A", UserEmail: "a@x.com", LinkedRepos: []string{}},
		},
	}
	var buf bytes.Buffer
	if err := listWorkspaces(&buf, s, "json"); err != nil {
		t.Fatal(err)
	}
	var got map[string]*shared.Workspace
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["id-a"] == nil || got["id-a"].Name != "alpha" {
		t.Fatalf("got %+v", got)
	}
}

func TestShowProfileTable(t *testing.T) {
	s := seedState(t, "id-1", "work", "Worker", "w@x.com", "repo-1", "myrepo", "/r")
	var buf bytes.Buffer
	if err := showWorkspace(&buf, s, "work", "table"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"name:         work", "id:           id-1", "user.name:    Worker", "linked repos:"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestShowProfileJSON(t *testing.T) {
	s := seedState(t, "id-1", "work", "Worker", "w@x.com", "", "", "")
	var buf bytes.Buffer
	if err := showWorkspace(&buf, s, "work", "json"); err != nil {
		t.Fatal(err)
	}
	var got struct {
		ID        string            `json:"id"`
		Workspace *shared.Workspace `json:"workspace"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "id-1" || got.Workspace.Name != "work" {
		t.Fatalf("got %+v", got)
	}
}

func TestShowProfileNotFound(t *testing.T) {
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(t.TempDir(), "state.json"))
	s, _ := shared.Load()
	var buf bytes.Buffer
	err := showWorkspace(&buf, s, "nope", "table")
	if !errors.Is(err, shared.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}
