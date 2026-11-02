package state

import (
	"fmt"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
)

func TestChangesContained(t *testing.T) {
	m := git.NewMemoryRunner()
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	if ChangesContained("", "a2") || ChangesContained("main", "a2") {
		t.Fatal("empty merge-base is not contained")
	}

	m.Outputs["merge-base main a2"] = "o\n"
	m.Outputs["rev-parse main^{tree}"] = "treeS\n"
	m.Outputs["merge-tree --write-tree --merge-base=o main a2"] = "treeS\n"
	if !ChangesContained("main", "a2") {
		t.Fatal("matching tree should be contained")
	}

	m.Outputs["merge-base main b1"] = "o"
	m.Outputs["merge-tree --write-tree --merge-base=o main b1"] = "other"
	if ChangesContained("main", "b1") {
		t.Fatal("different tree should not be contained")
	}

	m.Outputs["merge-base main bad"] = "o"
	m.FailOn["merge-tree --write-tree --merge-base=o main bad"] = fmt.Errorf("conflict")
	if ChangesContained("main", "bad") {
		t.Fatal("merge-tree failure should not be contained")
	}
}

func TestAlreadyAppliedBase(t *testing.T) {
	m := git.NewMemoryRunner()
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	if AlreadyAppliedBase("main") != "" {
		t.Fatal("missing merge-base should return empty")
	}

	m.Outputs["merge-base main HEAD"] = "o"
	m.Outputs["rev-list --reverse --first-parent o..HEAD"] = "a1\na2\nb1"
	m.Outputs["rev-parse main^{tree}"] = "treeS"
	m.Outputs["merge-tree --write-tree --merge-base=o main a1"] = "treeS"
	m.Outputs["merge-tree --write-tree --merge-base=o main a2"] = "treeS"
	m.Outputs["merge-tree --write-tree --merge-base=o main b1"] = "beyond"
	if got := AlreadyAppliedBase("main"); got != "a2" {
		t.Fatalf("got %q, want a2", got)
	}
}

func TestAlreadyAppliedBaseWhenMergeTreeFails(t *testing.T) {
	m := git.NewMemoryRunner()
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	m.Outputs["merge-base main HEAD"] = "o"
	m.Outputs["rev-list --reverse --first-parent o..HEAD"] = "a1"
	m.FailOn["merge-tree --write-tree --merge-base=o main a1"] = fmt.Errorf("unknown option")

	if got := AlreadyAppliedBase("main"); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}
