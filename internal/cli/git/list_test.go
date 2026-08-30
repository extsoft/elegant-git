package git

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/shared"
)

func TestListCommandOutput(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	m := git.NewMemoryRunner()
	m.GlobalConfig["user.name"] = "Global"
	git.Use(m)
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	s.Workspaces["p"] = &shared.Workspace{Name: "p", UserName: "P", UserEmail: "p@x.com", LinkedRepos: []string{}}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	cmd := newListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(nil)
	cmd.SetContext(context.Background())

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"version:", "shared memory:", "global git identity:"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(out, "profiles:\n") {
		t.Fatal("should not dump workspaces section")
	}
}
