package repo

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/shared"
)

func TestConfigureLocalGitInstallPreGlobalAcquired(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, _ := shared.Load()
	shared.SetAcquired(s, "1.0.0")
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	m := git.NewMemoryRunner()
	m.Outputs["config --local --get-regexp "+config.AcquiredKey] = config.AcquiredKey + " true"
	m.Outputs["config --local --get-regexp ^alias\\."] = "alias.start-work elegant work start"
	git.Use(m)

	if err := configureLocalGitInstallPre(); err != nil {
		t.Fatal(err)
	}
	assertConfigCall(t, m.Calls, "--local", "--unset", config.AcquiredKey)
	assertConfigCall(t, m.Calls, "--local", "--unset", "alias.start-work")
}

func TestConfigureLocalGitInstallPreLocalOnly(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	m := git.NewMemoryRunner()
	m.Outputs["config --local --get-regexp "+config.AcquiredKey] = ""
	m.Outputs["config --local --get-regexp ^alias\\."] = "alias.start-work elegant work start"
	git.Use(m)

	if err := configureLocalGitInstallPre(); err != nil {
		t.Fatal(err)
	}
	assertConfigCall(t, m.Calls, "--local", "--unset", "alias.start-work")
}

func TestConfigureLocalGitInstallPostGlobalAcquired(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, _ := shared.Load()
	shared.SetAcquired(s, "1.0.0")
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	m := git.NewMemoryRunner()
	git.Use(m)

	if err := configureLocalGitInstallPost(); err != nil {
		t.Fatal(err)
	}
	for _, c := range m.Calls {
		if len(c.Args) >= 3 && c.Args[0] == "config" && strings.HasPrefix(c.Args[2], "alias.") {
			t.Fatalf("unexpected alias config: %+v", c.Args)
		}
	}
}

func TestConfigureLocalGitInstallPostLocalOnly(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	m := git.NewMemoryRunner()
	git.Use(m)

	if err := configureLocalGitInstallPost(); err != nil {
		t.Fatal(err)
	}
	foundAlias := false
	for _, c := range m.Calls {
		if len(c.Args) >= 3 && c.Args[0] == "config" && strings.HasPrefix(c.Args[2], "alias.") {
			foundAlias = true
		}
	}
	if !foundAlias {
		t.Fatal("expected local aliases to be configured")
	}
}

func assertConfigCall(t *testing.T, calls []git.Call, scope, verb, key string) {
	t.Helper()
	for _, c := range calls {
		if len(c.Args) >= 4 && c.Args[0] == "config" && c.Args[1] == scope && c.Args[2] == verb && c.Args[3] == key {
			return
		}
	}
	t.Fatalf("missing config %s %s %s in %+v", scope, verb, key, calls)
}
