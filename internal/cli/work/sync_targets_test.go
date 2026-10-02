package work

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
)

func TestSyncBranchCompletePinnedOrderAndFetch(t *testing.T) {
	const currentBranch = "feat"
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(gitDir, "elegant-git", "state.json"))

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	m.Repo.CurrentBranch = currentBranch
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["rev-parse --verify --quiet main"] = "abc"
	m.Outputs["rev-parse --verify --quiet origin/main"] = "abc"
	m.Outputs["rev-parse --verify --quiet --abbrev-ref --branches=refs/heads "+currentBranch] = currentBranch
	m.Outputs["rev-parse --abbrev-ref "+currentBranch+"@{upstream}"] = "origin/feat"
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "feat\torigin/feat\nmain\t"
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/feat\norigin/main\nupstream/feat"
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{currentBranch: "main"},
	}); err != nil {
		t.Fatal(err)
	}
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	choices, err := syncBranchComplete(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(choices) < 2 {
		t.Fatalf("choices=%v", choices)
	}
	if choices[0].Value != "origin/main" || choices[0].Description != "Source branch." {
		t.Fatalf("source row: %+v", choices[0])
	}
	if choices[1].Value != "origin/feat" || choices[1].Description != "This branch's upstream." {
		t.Fatalf("upstream row: %+v", choices[1])
	}
	seen := map[string]int{}
	for _, c := range choices {
		seen[c.Value]++
	}
	for val, n := range seen {
		if n > 1 {
			t.Fatalf("duplicate %q", val)
		}
	}
	for _, c := range choices {
		if c.Value == currentBranch {
			t.Fatalf("listed local current branch %q", c.Value)
		}
	}
	var keptOtherRemote bool
	for _, c := range choices {
		if c.Value == "upstream/feat" {
			keptOtherRemote = true
		}
		if c.Value == "origin/"+currentBranch && c.Description != "This branch's upstream." {
			t.Fatalf("origin/%s listed outside the upstream pin: %+v", currentBranch, c)
		}
	}
	if !keptOtherRemote {
		t.Fatalf("dropped other remote of the current branch, choices=%v", choices)
	}

	var fetchIdx, headsIdx int
	for i, call := range m.Calls {
		if len(call.Args) > 0 && call.Args[0] == "fetch" {
			if fetchIdx == 0 {
				fetchIdx = i + 1
			}
		}
		if strings.Join(call.Args, " ") == "for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads" {
			if headsIdx == 0 {
				headsIdx = i + 1
			}
		}
	}
	if fetchIdx == 0 || headsIdx == 0 || fetchIdx >= headsIdx {
		t.Fatalf("expected fetch before for-each-ref heads, calls=%+v", m.Calls)
	}
	if fetchCallCount(m) != 1 {
		t.Fatalf("expected one fetch, calls=%+v", m.Calls)
	}
	for _, c := range m.Calls {
		if len(c.Args) >= 2 && c.Args[0] == "fetch" && c.Args[1] == "--all" {
			t.Fatalf("unexpected fetch --all, calls=%+v", m.Calls)
		}
	}
}

func TestSyncBranchCompleteSkipsMissingDefault(t *testing.T) {
	const currentBranch = "feat"
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(gitDir, "elegant-git", "state.json"))

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	m.Repo.CurrentBranch = currentBranch
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["rev-parse --verify --quiet --abbrev-ref --branches=refs/heads "+currentBranch] = currentBranch
	m.Outputs["rev-parse --verify --quiet origin/master"] = "abc"
	m.FailOn["rev-parse --verify --quiet origin/main"] = fmt.Errorf("missing")
	m.Outputs["rev-parse --abbrev-ref "+currentBranch+"@{upstream}"] = "origin/feat"
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "feat\t\nmaster\t"
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/master\norigin/feat"
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{currentBranch: "master"},
	}); err != nil {
		t.Fatal(err)
	}
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	choices, err := syncBranchComplete(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range choices {
		if c.Value == "origin/main" {
			t.Fatalf("listed missing default %+v", c)
		}
	}
	if len(choices) == 0 || choices[0].Value != "origin/master" || choices[0].Description != "Source branch." {
		t.Fatalf("source row: %+v", choices)
	}
}
