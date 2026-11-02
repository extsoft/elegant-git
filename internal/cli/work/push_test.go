package work

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
)

func TestPushRemoteBranch(t *testing.T) {
	tests := []struct {
		local, arg, upstream, want string
	}{
		{"feature1", "", "", "feature1"},
		{"feature1", "", "origin/feature1", "feature1"},
		{"feature1", "other", "", "other"},
		{"OPS-2162", "", "origin/main", "OPS-2162"},
		{"123", "", "some-remote/feature/123", "feature/123"},
	}
	for _, tc := range tests {
		got := pushRemoteBranch(tc.local, tc.arg, tc.upstream)
		if got != tc.want {
			t.Errorf("pushRemoteBranch(%q, %q, %q) = %q, want %q", tc.local, tc.arg, tc.upstream, got, tc.want)
		}
	}
}

func TestPushLogicRebaseOntoSkipsAppliedPrefix(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(gitDir, "elegant-git", "state.json"))
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	m.Repo.CurrentBranch = "B"
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["rev-parse --verify --quiet main"] = "abc"
	m.Outputs["rev-parse --verify --quiet origin/main"] = "abc"
	m.Outputs["rev-parse --verify --quiet --abbrev-ref --branches=refs/heads B"] = "B"
	m.FailOn["rev-parse --abbrev-ref main@{upstream}"] = fmt.Errorf("no upstream")
	m.FailOn["rev-parse --abbrev-ref B@{upstream}"] = fmt.Errorf("no upstream")
	if err := memrepo.Save(gitDir, &memrepo.State{DefaultBranch: "main"}); err != nil {
		t.Fatal(err)
	}
	stubAppliedPrefix(m, "origin/main", "a2", []string{"a1", "a2", "b1"})
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	if err := pushLogic("B", ""); err != nil {
		t.Fatal(err)
	}
	if !hasGitCall(m, "rebase", "--onto", "origin/main", "a2") {
		t.Fatalf("calls=%+v", m.Calls)
	}
}
