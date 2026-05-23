package workflows

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bees-hive/elegant-git/internal/cmdid"
)

func TestWorkflowsDirectoryInitRepository(t *testing.T) {
	old := repoRootFunc
	repoRootFunc = func() string { return "." }
	defer func() { repoRootFunc = old }()

	initID := cmdid.ID{Command: "repo", Action: "init"}
	dir, err := WorkflowsDirectory("common", initID)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(".config", "elegant-git", "hooks")
	if dir != want {
		t.Fatalf("dir = %q, want %q", dir, want)
	}
	cloneID := cmdid.ID{Command: "repo", Action: "clone"}
	personal, err := WorkflowsDirectory("personal", cloneID)
	if err != nil {
		t.Fatal(err)
	}
	wantPersonal := filepath.Join(".git", ".config", "elegant-git", "hooks")
	if personal != wantPersonal {
		t.Fatalf("personal = %q, want %q", personal, wantPersonal)
	}
}

func TestWorkflowsFileJoin(t *testing.T) {
	old := repoRootFunc
	repoRootFunc = func() string { return "." }
	defer func() { repoRootFunc = old }()

	initID := cmdid.ID{Command: "repo", Action: "init"}
	path, err := WorkflowsFile("common", initID, "ahead")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(".config", "elegant-git", "hooks", "repo-init-ahead")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestPrefixSkipsInitAndClone(t *testing.T) {
	if Prefix(cmdid.ID{Command: "repo", Action: "init"}) != "" {
		t.Fatal("repo init should have empty prefix")
	}
	if Prefix(cmdid.ID{Command: "repo", Action: "clone"}) != "" {
		t.Fatal("repo clone should have empty prefix")
	}
}

func TestSkipDisablesHooks(t *testing.T) {
	Skip = true
	ctx := context.Background()
	RunAhead(ctx, cmdid.ID{Command: "work", Action: "start"})
	RunAfter(ctx, cmdid.ID{Command: "work", Action: "start"})
	Skip = false
}
