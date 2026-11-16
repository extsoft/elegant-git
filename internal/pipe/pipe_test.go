package pipe

import (
	"fmt"
	"path/filepath"
	"strings"
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

func TestStashPipePopsWhenSourceBranchHasApostrophe(t *testing.T) {
	m := setupBranchPipeTest(t, "main")
	name := "feat/don't"
	stubLocalBranch(m, name)
	id := cmdid.ID{Command: "work", Action: "accept"}
	msg := "eg work.accept auto-stash: WIP in '" + name + "' branch on 2020-01-01T00:00:00"
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

func TestStashPipeKeepsAssociationWhenPopFails(t *testing.T) {
	m := setupBranchPipeTest(t, "main")
	stubLocalBranch(m, "feat")
	id := cmdid.ID{Command: "work", Action: "accept"}
	msg := "eg work.accept auto-stash: WIP in 'feat' branch on 2020-01-01T00:00:00"
	if err := cmdmem.Set(id, cmdmem.FieldStash, msg); err != nil {
		t.Fatal(err)
	}
	m.Outputs["stash list --grep="+msg+" --format=%gd"] = "stash@{0}"
	m.FailOn["stash pop stash@{0}"] = fmt.Errorf("conflict")

	err := StashPipe(id, func() error { return nil })
	if err == nil {
		t.Fatal("expected restore error")
	}
	got, gerr := cmdmem.Get(id, cmdmem.FieldStash)
	if gerr != nil || got != msg {
		t.Fatalf("association %q err=%v", got, gerr)
	}
	if hasStashDrop(m) {
		t.Fatalf("dropped a stash, calls=%v", m.Calls)
	}
}

func TestStashPipeMissingLegacyBackupDropsNothing(t *testing.T) {
	m := setupBranchPipeTest(t, "main")
	stubLocalBranch(m, "feat")
	id := cmdid.ID{Command: "work", Action: "accept"}
	msg := "eg work.accept auto-stash: WIP in 'feat' branch on 2020-01-01T00:00:00"
	if err := cmdmem.Set(id, cmdmem.FieldStash, msg); err != nil {
		t.Fatal(err)
	}
	m.Outputs["stash list --grep="+msg+" --format=%gd"] = ""
	m.Outputs["stash list --format=%gd"] = "stash@{0}"

	err := StashPipe(id, func() error { return nil })
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err=%v", err)
	}
	if hasStashPop(m, "stash@{0}") || hasStashDrop(m) {
		t.Fatalf("changed another stash, calls=%v", m.Calls)
	}
	got, _ := cmdmem.Get(id, cmdmem.FieldStash)
	if got != msg {
		t.Fatalf("association %q", got)
	}
}

func TestStashPipeReportsBothBodyAndRestoreErrors(t *testing.T) {
	m := setupBranchPipeTest(t, "main")
	stubLocalBranch(m, "feat")
	id := cmdid.ID{Command: "work", Action: "switch"}
	msg := "eg work.switch auto-stash: WIP in 'feat' branch on 2020-01-01T00:00:00"
	if err := cmdmem.Set(id, cmdmem.FieldStash, msg); err != nil {
		t.Fatal(err)
	}
	m.Outputs["stash list --grep="+msg+" --format=%gd"] = "stash@{0}"
	m.FailOn["stash pop stash@{0}"] = fmt.Errorf("conflict")

	err := StashPipeRestoring(id, func() error { return fmt.Errorf("body failed") })
	if err == nil || !strings.Contains(err.Error(), "body failed") || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("err=%v", err)
	}
}

func TestHasChangesRefreshesIndexFirst(t *testing.T) {
	m := setupBranchPipeTest(t, "main")

	HasChanges()

	if len(m.Calls) == 0 || len(m.Calls[0].Args) < 1 || m.Calls[0].Args[0] != "update-index" {
		t.Fatalf("expected index refresh before diff-index, calls=%v", m.Calls)
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

func hasStashDrop(m *git.MemoryRunner) bool {
	for _, c := range m.Calls {
		if len(c.Args) >= 2 && c.Args[0] == "stash" && c.Args[1] == "drop" {
			return true
		}
	}
	return false
}
