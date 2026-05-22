package workflows

import (
	"path/filepath"
	"testing"
)

func TestWorkflowsDirectoryInitRepository(t *testing.T) {
	dir, err := WorkflowsDirectory("common", "init-repository")
	if err != nil {
		t.Fatal(err)
	}
	if dir != ".workflows" {
		t.Fatalf("dir = %q, want .workflows", dir)
	}
	personal, err := WorkflowsDirectory("personal", "clone-repository")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(".git", ".workflows")
	if personal != want {
		t.Fatalf("personal = %q, want %q", personal, want)
	}
}

func TestWorkflowsFileJoin(t *testing.T) {
	path, err := WorkflowsFile("common", "init-repository", "ahead")
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(".workflows", "start-work-ahead") {
		// init uses command name in filename
	}
	if path != filepath.Join(".workflows", "init-repository-ahead") {
		t.Fatalf("path = %q", path)
	}
}

func TestPrefixSkipsInitAndClone(t *testing.T) {
	if Prefix("init-repository") != "" {
		t.Fatal("init-repository should have empty prefix")
	}
	if Prefix("clone-repository") != "" {
		t.Fatal("clone-repository should have empty prefix")
	}
}

func TestSkipDisablesHooks(t *testing.T) {
	Skip = true
	RunAhead("start-work")
	RunAfter("start-work")
	Skip = false
}
