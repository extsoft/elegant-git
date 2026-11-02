package work

import (
	"fmt"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
)

func TestSkipMessage(t *testing.T) {
	if got := skipMessage(1, "main"); got != "Skipping 1 commit already in main." {
		t.Fatalf("got %q", got)
	}
	if got := skipMessage(2, "origin/main"); got != "Skipping 2 commits already in origin/main." {
		t.Fatalf("got %q", got)
	}
}

func TestRebaseOntoSkipsAppliedPrefix(t *testing.T) {
	m := git.NewMemoryRunner()
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	stubAppliedPrefix(m, "origin/main", "a2", []string{"a1", "a2", "b1"})

	if err := rebaseOnto("origin/main"); err != nil {
		t.Fatal(err)
	}
	if !hasGitCall(m, "rebase", "--onto", "origin/main", "a2") {
		t.Fatalf("calls=%+v", m.Calls)
	}
}

func TestRebaseOntoPlainWhenNothingApplied(t *testing.T) {
	m := git.NewMemoryRunner()
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	if err := rebaseOnto("origin/main"); err != nil {
		t.Fatal(err)
	}
	if !hasGitCall(m, "rebase", "origin/main") {
		t.Fatalf("calls=%+v", m.Calls)
	}
	if hasGitCall(m, "rebase", "--onto") {
		t.Fatalf("unexpected --onto, calls=%+v", m.Calls)
	}
}

func TestRebaseOntoPlainWhenMergeTreeFails(t *testing.T) {
	m := git.NewMemoryRunner()
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	m.Outputs["merge-base origin/main HEAD"] = "mb"
	m.Outputs["rev-list --reverse --first-parent mb..HEAD"] = "a1"
	m.FailOn["merge-tree --write-tree --merge-base=mb origin/main a1"] = fmt.Errorf("unknown option")

	if err := rebaseOnto("origin/main"); err != nil {
		t.Fatal(err)
	}
	if !hasGitCall(m, "rebase", "origin/main") || hasGitCall(m, "rebase", "--onto") {
		t.Fatalf("calls=%+v", m.Calls)
	}
}

func stubAppliedPrefix(m *git.MemoryRunner, target, base string, commits []string) {
	m.Outputs["merge-base "+target+" HEAD"] = "mb"
	joined := ""
	for i, c := range commits {
		if i > 0 {
			joined += "\n"
		}
		joined += c
		tree := "beyond"
		if c == base || c == "a1" {
			tree = "applied"
		}
		m.Outputs["merge-tree --write-tree --merge-base=mb "+target+" "+c] = tree
	}
	m.Outputs["rev-list --reverse --first-parent mb..HEAD"] = joined
	m.Outputs["rev-parse "+target+"^{tree}"] = "applied"
	m.Outputs["rev-list --count mb.."+base] = "2"
}
