package state

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
)

func TestLocalBranchNames(t *testing.T) {
	mem := git.NewMemoryRunner()
	mem.Outputs["for-each-ref --format=%(refname:lstrip=2) refs/heads"] = "main\n\n feature/x \n"
	git.Use(mem)
	got, err := LocalBranchNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "main" || got[1] != "feature/x" {
		t.Fatalf("got %#v", got)
	}

	mem.FailOn["for-each-ref --format=%(refname:lstrip=2) refs/heads"] = fmt.Errorf("not a git repository")
	if _, err := LocalBranchNames(); err == nil || err.Error() != "not a git repository" {
		t.Fatalf("err=%v", err)
	}
}

func TestUpstreamOfEmptyWhenUnset(t *testing.T) {
	mem := git.NewMemoryRunner()
	mem.FailOn["rev-parse --abbrev-ref 1635@{upstream}"] = fmt.Errorf("fatal: no upstream configured for branch '1635'")
	git.Use(mem)
	got := UpstreamOf("1635")
	if got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestUpstreamOfReturnsTrackingBranch(t *testing.T) {
	mem := git.NewMemoryRunner()
	mem.Outputs["rev-parse --abbrev-ref feature@{upstream}"] = "origin/feature\n"
	git.Use(mem)
	got := UpstreamOf("feature")
	if got != "origin/feature" {
		t.Fatalf("got %q", got)
	}
}

func TestLocalBranchNamesRealRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	run("config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("o\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "f")
	run("commit", "-q", "-m", "o")
	run("branch", "v1")
	run("tag", "v1")
	run("branch", "rel")
	run("tag", "rel")
	run("pack-refs", "--all")
	if _, err := os.Stat(filepath.Join(dir, ".git", "refs", "heads", "v1")); !os.IsNotExist(err) {
		t.Fatalf("loose ref refs/heads/v1 still present: %v", err)
	}

	t.Chdir(dir)
	git.Use(git.RealRunner{})
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	names, err := LocalBranchNames()
	if err != nil {
		t.Fatal(err)
	}
	if got := git.NewBranchNameProblem("v1", names); got != "branch 'v1' already exists" {
		t.Fatalf("v1: %q names=%v", got, names)
	}
	if got := git.NewBranchNameProblem("rel/x", names); got != "branch 'rel/x' conflicts with existing branch 'rel'" {
		t.Fatalf("rel/x: %q names=%v", got, names)
	}
	if got := git.NewBranchNameProblem("main", names); got != "branch 'main' already exists" {
		t.Fatalf("packed main: %q names=%v", got, names)
	}
}
