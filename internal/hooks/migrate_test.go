package hooks

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/extsoft/elegant-git/internal/runtime"
)

func TestMigrateMovesExtraFilesAndRemovesWorkflows(t *testing.T) {
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

	ws := runtime.RepoLayout{RepoRoot: root}
	newPaths, oldPaths, err := Migrate(ws, false, false, io.Discard)
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

func TestHasLegacy(t *testing.T) {
	root := t.TempDir()
	ws := runtime.RepoLayout{RepoRoot: root}
	if HasLegacy(ws, false) {
		t.Fatal("expected no legacy dir")
	}
	if err := os.MkdirAll(filepath.Join(root, ".workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !HasLegacy(ws, false) {
		t.Fatal("expected legacy dir")
	}
}
