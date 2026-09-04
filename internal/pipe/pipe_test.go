package pipe

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/git"
	cmdmem "github.com/extsoft/elegant-git/internal/memory/cmd"
)

func TestBranchPipeRestoresWhenPreviousExists(t *testing.T) {
	m := setupBranchPipeTest(t, "feat")
	stubLocalBranch(m, "feat")
	id := cmdid.ID{Command: "work", Action: "accept"}

	if err := BranchPipe(id, func() error {
		m.Repo.CurrentBranch = "main"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !hasCheckout(m, "feat") {
		t.Fatalf("missing checkout feat, calls=%v", m.Calls)
	}
}

func TestBranchPipeSkipsRestoreWhenPreviousGone(t *testing.T) {
	m := setupBranchPipeTest(t, "feat")
	stubLocalBranch(m, "feat")
	id := cmdid.ID{Command: "work", Action: "accept"}

	if err := BranchPipe(id, func() error {
		delete(m.Outputs, localBranchExistsKey("feat"))
		m.FailOn[localBranchExistsKey("feat")] = fmt.Errorf("unknown")
		m.Repo.CurrentBranch = "main"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if hasCheckout(m, "feat") {
		t.Fatalf("unexpected checkout feat, calls=%v", m.Calls)
	}
}

func TestStashPipePopsWhenSourceBranchExists(t *testing.T) {
	m := setupBranchPipeTest(t, "main")
	stubLocalBranch(m, "feat")
	id := cmdid.ID{Command: "work", Action: "accept"}
	msg := "eg work.accept auto-stash: WIP in 'feat' branch on 2020-01-01T00:00:00"
	if err := cmdmem.Set(id, cmdmem.FieldStash, msg); err != nil {
		t.Fatal(err)
	}
	m.Outputs["stash list --grep="+msg+" --format=%gd"] = "stash@{0}"

	if err := StashPipe(id, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if !hasStashPop(m, "stash@{0}") {
		t.Fatalf("missing stash pop, calls=%v", m.Calls)
	}
}

func TestStashPipeSkipsPopWhenSourceBranchGone(t *testing.T) {
	m := setupBranchPipeTest(t, "main")
	id := cmdid.ID{Command: "work", Action: "accept"}
	msg := "eg work.accept auto-stash: WIP in 'feat' branch on 2020-01-01T00:00:00"
	if err := cmdmem.Set(id, cmdmem.FieldStash, msg); err != nil {
		t.Fatal(err)
	}
	m.Outputs["stash list --grep="+msg+" --format=%gd"] = "stash@{0}"
	m.FailOn[localBranchExistsKey("feat")] = fmt.Errorf("unknown")

	if err := StashPipe(id, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if hasStashPop(m, "stash@{0}") {
		t.Fatalf("unexpected stash pop, calls=%v", m.Calls)
	}
}

func setupBranchPipeTest(t *testing.T, current string) *git.MemoryRunner {
	t.Helper()
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	m := git.NewMemoryRunner()
	m.Repo.CurrentBranch = current
	m.Outputs["rev-parse --git-dir"] = gitDir
	git.Use(m)
	return m
}

func stubLocalBranch(m *git.MemoryRunner, name string) {
	m.Outputs[localBranchExistsKey(name)] = name
}

func localBranchExistsKey(name string) string {
	return "rev-parse --verify --quiet --abbrev-ref --branches=refs/heads " + name
}

func hasCheckout(m *git.MemoryRunner, branch string) bool {
	for _, c := range m.Calls {
		if len(c.Args) == 2 && c.Args[0] == "checkout" && c.Args[1] == branch {
			return true
		}
	}
	return false
}

func hasStashPop(m *git.MemoryRunner, sid string) bool {
	for _, c := range m.Calls {
		if len(c.Args) == 3 && c.Args[0] == "stash" && c.Args[1] == "pop" && c.Args[2] == sid {
			return true
		}
	}
	return false
}
