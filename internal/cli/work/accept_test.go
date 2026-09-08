package work

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

func TestAcceptLogicContinuesHelperRebaseWithoutBranch(t *testing.T) {
	m := git.NewMemoryRunner()
	git.Use(m)
	orig := isAcceptHelperRebase
	isAcceptHelperRebase = func() bool { return true }
	defer func() { isAcceptHelperRebase = orig }()

	var branch string
	spec := acceptSpec(&branch)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	if err := acceptLogic(cmd, nil, spec); err != nil {
		t.Fatal(err)
	}
	if len(m.Calls) != 1 || m.Calls[0].Args[0] != "rebase" || m.Calls[0].Args[1] != "--continue" {
		t.Fatalf("calls=%v", m.Calls)
	}
	if branch != "" {
		t.Fatalf("branch=%q", branch)
	}
}

func TestAcceptLogicDeletesLocalWorkBranch(t *testing.T) {
	m := setupAcceptTest(t)
	stubLocalBranch(m, "feat")
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "feat\nmain"

	if err := runAcceptLogic(t, "feat"); err != nil {
		t.Fatal(err)
	}
	got := forceDeletedBranches(m)
	if !slices.Contains(got, acceptWorkBranch) {
		t.Fatalf("missing helper delete, got %v", got)
	}
	if !slices.Contains(got, "feat") {
		t.Fatalf("missing local work branch delete, got %v", got)
	}
}

func TestAcceptLogicDeletesLocalWorkBranchAfterPush(t *testing.T) {
	m := setupAcceptTest(t)
	m.Repo.Remotes = []string{"origin"}
	stubLocalBranch(m, "feat")
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "feat\nmain"

	if err := runAcceptLogic(t, "feat"); err != nil {
		t.Fatal(err)
	}
	pushIdx := callIndex(m, "push", "origin", "main:main")
	featIdx := callIndex(m, "branch", "--delete", "--force", "feat")
	if pushIdx < 0 || featIdx < 0 {
		t.Fatalf("push=%d feat-delete=%d calls=%v", pushIdx, featIdx, m.Calls)
	}
	if featIdx < pushIdx {
		t.Fatalf("local delete before push, calls=%v", m.Calls)
	}
}

func TestAcceptLogicKeepsLocalWorkBranchWhenPushFails(t *testing.T) {
	m := setupAcceptTest(t)
	m.Repo.Remotes = []string{"origin"}
	stubLocalBranch(m, "feat")
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "feat\nmain"
	m.FailOn["push origin main:main"] = fmt.Errorf("rejected")

	if err := runAcceptLogic(t, "feat"); err == nil {
		t.Fatal("expected push error")
	}
	got := forceDeletedBranches(m)
	if !slices.Contains(got, acceptWorkBranch) {
		t.Fatalf("missing helper delete, got %v", got)
	}
	if slices.Contains(got, "feat") {
		t.Fatalf("deleted local work branch after push failure, got %v", got)
	}
}

func TestAcceptLogicSucceedsWhenLocalWorkBranchDeleteFails(t *testing.T) {
	m := setupAcceptTest(t)
	stubLocalBranch(m, "feat")
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "feat\nmain"
	m.FailOn["branch --delete --force feat"] = fmt.Errorf("used by worktree")

	if err := runAcceptLogic(t, "feat"); err != nil {
		t.Fatal(err)
	}
	if callIndex(m, "branch", "--delete", "--force", "feat") < 0 {
		t.Fatalf("missing local work branch delete attempt, calls=%v", m.Calls)
	}
}

func TestAcceptLogicDoesNotDeleteHelperTwice(t *testing.T) {
	m := setupAcceptTest(t)
	stubLocalBranch(m, acceptWorkBranch)
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = acceptWorkBranch + "\nmain"

	if err := runAcceptLogic(t, acceptWorkBranch); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, name := range forceDeletedBranches(m) {
		if name == acceptWorkBranch {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("helper deletes=%d, want 1, calls=%v", n, m.Calls)
	}
}

func TestAcceptLogicDoesNotDeleteProtectedLocalBranch(t *testing.T) {
	m := setupAcceptTest(t)
	stubLocalBranch(m, "main")
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "main"

	if err := runAcceptLogic(t, "main"); err != nil {
		t.Fatal(err)
	}
	got := forceDeletedBranches(m)
	if !slices.Contains(got, acceptWorkBranch) {
		t.Fatalf("missing helper delete, got %v", got)
	}
	if slices.Contains(got, "main") {
		t.Fatalf("deleted protected default, got %v", got)
	}
}

func TestAcceptLogicDoesNotDeleteProtectedNonDefault(t *testing.T) {
	m := setupAcceptTest(t)
	gitDir := m.Outputs["rev-parse --git-dir"]
	if err := memrepo.Save(gitDir, &memrepo.State{
		DefaultBranch:     "main",
		ProtectedBranches: []string{"main", "release"},
	}); err != nil {
		t.Fatal(err)
	}
	stubLocalBranch(m, "release")
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "release\nmain"

	if err := runAcceptLogic(t, "release"); err != nil {
		t.Fatal(err)
	}
	got := forceDeletedBranches(m)
	if slices.Contains(got, "release") {
		t.Fatalf("deleted protected branch, got %v", got)
	}
}

func TestAcceptLogicDoesNotDeleteRemoteOnlyWorkBranch(t *testing.T) {
	m := setupAcceptTest(t)
	m.Repo.Remotes = []string{"origin"}
	m.FailOn["rev-parse --verify --quiet --abbrev-ref --branches=refs/heads origin/feat"] = fmt.Errorf("unknown")
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "main"
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/feat"

	if err := runAcceptLogic(t, "origin/feat"); err != nil {
		t.Fatal(err)
	}
	got := forceDeletedBranches(m)
	if !slices.Contains(got, acceptWorkBranch) {
		t.Fatalf("missing helper delete, got %v", got)
	}
	if slices.Contains(got, "origin/feat") || slices.Contains(got, "feat") {
		t.Fatalf("deleted remote-only work branch, got %v", got)
	}
}

func setupAcceptTest(t *testing.T) *git.MemoryRunner {
	t.Helper()
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(gitDir, "elegant-git", "state.json"))
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	m.Outputs["rev-parse --verify --quiet main"] = "abc"
	m.FailOn["rev-parse --abbrev-ref main@{upstream}"] = fmt.Errorf("no upstream")
	if err := memrepo.Save(gitDir, &memrepo.State{
		DefaultBranch:     "main",
		ProtectedBranches: []string{"main"},
	}); err != nil {
		t.Fatal(err)
	}
	git.Use(m)
	return m
}

func stubLocalBranch(m *git.MemoryRunner, name string) {
	m.Outputs["rev-parse --verify --quiet --abbrev-ref --branches=refs/heads "+name] = name
	m.FailOn["rev-parse --abbrev-ref "+name+"@{upstream}"] = fmt.Errorf("no upstream")
}

func runAcceptLogic(t *testing.T, branch string) error {
	t.Helper()
	var got string
	spec := acceptSpec(&got)
	cmd := &cobra.Command{}
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	return acceptLogic(cmd, []string{branch}, spec)
}

func forceDeletedBranches(m *git.MemoryRunner) []string {
	var names []string
	for _, c := range m.Calls {
		if len(c.Args) == 4 && c.Args[0] == "branch" && c.Args[1] == "--delete" && c.Args[2] == "--force" {
			names = append(names, c.Args[3])
		}
	}
	return names
}

func callIndex(m *git.MemoryRunner, args ...string) int {
	for i, c := range m.Calls {
		if slices.Equal(c.Args, args) {
			return i
		}
	}
	return -1
}
