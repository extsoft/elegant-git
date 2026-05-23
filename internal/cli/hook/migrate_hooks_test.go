package hook

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bees-hive/elegant-git/internal/runtime"
)

func TestMigrateHooksMovesExtraFilesAndRemovesWorkflows(t *testing.T) {
	root := t.TempDir()
	legacyDir := filepath.Join(root, ".workflows")
	hooksDir := filepath.Join(root, ".config", "elegant-git", "hooks")
	if err := os.MkdirAll(legacyDir, 0o750); err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(legacyDir, "installation-workflows.bash")
	if err := os.WriteFile(extra, []byte("#!/bin/sh\n"), 0o750); err != nil {
		t.Fatal(err)
	}
	legacyHook := filepath.Join(legacyDir, "start-work-ahead")
	if err := os.WriteFile(legacyHook, []byte("echo ahead\n"), 0o700); err != nil {
		t.Fatal(err)
	}

	ws := runtime.Workspace{RepoRoot: root}
	newPaths, oldPaths, err := MigrateHooks(ws, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(newPaths) < 2 || len(oldPaths) < 2 {
		t.Fatalf("paths: new=%v old=%v", newPaths, oldPaths)
	}
	if _, err := os.Stat(legacyDir); !os.IsNotExist(err) {
		t.Fatal(".workflows should be removed")
	}
	if _, err := os.Stat(filepath.Join(hooksDir, "work-start-ahead")); err != nil {
		t.Fatal("canonical hook missing")
	}
	moved := filepath.Join(hooksDir, "installation-workflows.bash")
	info, err := os.Stat(moved)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o750 {
		t.Fatalf("mode = %o want 750", info.Mode().Perm())
	}
}
