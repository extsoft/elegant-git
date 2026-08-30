package hook

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/runtime"
	"github.com/extsoft/elegant-git/internal/text"
)

func TestListEmptyWorkspace(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	if err := listHooks(runtime.RepoLayout{RepoRoot: root}, &buf); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("got %q", buf.String())
	}
}

func TestListFindsHookFiles(t *testing.T) {
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
	if err := listHooks(runtime.RepoLayout{RepoRoot: root}, &buf); err != nil {
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
	if len(lines) < len(files) {
		t.Fatalf("got %d lines %v want at least %d paths", len(lines), lines, len(files))
	}
}

func TestListRunsStatusAndListHooks(t *testing.T) {
	root := t.TempDir()
	hooksDir := filepath.Join(root, ".config", "elegant-git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"hook-list-ahead", "hook-status-ahead"} {
		path := filepath.Join(hooksDir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\necho RAN:"+name+"\n"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --show-toplevel"] = root
	git.Use(m)

	var textBuf bytes.Buffer
	text.SetOutput(&textBuf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })

	cmd := newListCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetContext(context.Background())
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := textBuf.String()
	for _, name := range []string{"hook-list-ahead", "hook-status-ahead"} {
		if !strings.Contains(got, name) {
			t.Errorf("did not run %s; output:\n%s", name, got)
		}
	}
}
