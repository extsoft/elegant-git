package hook

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/runtime"
)

func TestStatusListEmptyWorkspace(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	if err := statusList(runtime.RepoLayout{RepoRoot: root}, &buf); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("got %q", buf.String())
	}
}

func TestStatusListFindsHookFiles(t *testing.T) {
	root := t.TempDir()
	hooksDir := filepath.Join(root, ".config", "elegant-git", "hooks")
	workflowsDir := filepath.Join(root, ".workflows")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(workflowsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"hook-status-ahead":    filepath.Join(hooksDir, "hook-status-ahead"),
		"hook-list-ahead":      filepath.Join(hooksDir, "hook-list-ahead"),
		"work-start-after":     filepath.Join(hooksDir, "work-start-after"),
		"show-workflows-ahead": filepath.Join(workflowsDir, "show-workflows-ahead"),
	}
	for _, path := range files {
		if err := os.WriteFile(path, []byte("# hook\n"), 0o700); err != nil {
			t.Fatal(err)
		}
	}

	var buf bytes.Buffer
	if err := statusList(runtime.RepoLayout{RepoRoot: root}, &buf); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	seen := map[string]bool{}
	for _, line := range lines {
		seen[line] = true
	}
	for name, path := range files {
		if !seen[path] {
			t.Errorf("missing %s at %s; got lines %v", name, path, lines)
		}
	}
	// hook-status is scanned from both personal and common dirs (same flat path).
	if len(lines) < len(files) {
		t.Fatalf("got %d lines %v want at least %d paths", len(lines), lines, len(files))
	}
}
