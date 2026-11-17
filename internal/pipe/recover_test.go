package pipe

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/git"
	cmdmem "github.com/extsoft/elegant-git/internal/memory/cmd"
	"github.com/extsoft/elegant-git/internal/recovery"
)

func TestPrepareRefusesGitOlderThanFloor(t *testing.T) {
	m := setupRecoveryRepo(t)
	m.Outputs["version"] = "git version 2.12.0"
	m.FailOn["diff-index --quiet HEAD"] = fmt.Errorf("dirty")

	err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		t.Fatal("body ran")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "Git "+recovery.MinGitVersion) {
		t.Fatalf("err=%v", err)
	}
	if hasStashPush(m) {
		t.Fatalf("stashed on an old git, calls=%v", m.Calls)
	}
}

func TestJournalWriteFailureDoesNotStash(t *testing.T) {
	m := setupRecoveryRepo(t)
	m.Outputs["version"] = "git version 2.54.0"
	m.Outputs["symbolic-ref --quiet HEAD"] = "refs/heads/main"
	m.FailOn["diff-index --quiet HEAD"] = fmt.Errorf("dirty")
	common := m.Outputs["rev-parse --git-common-dir"]
	if err := os.MkdirAll(filepath.Join(common, "elegant-git", "recovery.json"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		t.Fatal("body ran")
		return nil
	})
	if err == nil {
		t.Fatal("expected journal error")
	}
	if hasStashPush(m) {
		t.Fatalf("stashed without a journal, calls=%v", m.Calls)
	}
}

func TestApplyingPhaseDoesNotApplyAgain(t *testing.T) {
	m := setupRecoveryRepo(t)
	gitDir := m.Outputs["rev-parse --git-dir"]
	common := m.Outputs["rev-parse --git-common-dir"]
	id := cmdid.ID{Command: "work", Action: "sync"}
	op := recovery.Op{
		ID: "abc", Command: id.String(), Worktree: gitDir, Source: "main",
		Phase: recovery.PhaseApplying, StashOID: "deadbeef", Ref: recovery.RefName("abc"),
		Message: recovery.Message(id.String(), "abc"),
	}
	if err := recovery.Save(common, recovery.File{Ops: []recovery.Op{op}}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	err := StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "will not be applied again") {
		t.Fatalf("err=%v", err)
	}
	if calls != 0 || hasApply(m) {
		t.Fatalf("calls=%d git=%v", calls, m.Calls)
	}
}

func TestFailedPushRemovesPreparingRecord(t *testing.T) {
	m := setupRecoveryRepo(t)
	m.Outputs["version"] = "git version 2.54.0"
	m.Outputs["symbolic-ref --quiet HEAD"] = "refs/heads/main"
	m.FailOn["diff-index --quiet HEAD"] = fmt.Errorf("dirty")
	restoreID := recovery.NewID
	recovery.NewID = func() (string, error) { return "abc", nil }
	t.Cleanup(func() { recovery.NewID = restoreID })
	m.FailOn["stash push --message eg work.sync auto-stash abc"] = fmt.Errorf("index.lock")
	common := m.Outputs["rev-parse --git-common-dir"]

	err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		t.Fatal("body ran")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "index.lock") {
		t.Fatalf("err=%v", err)
	}
	f, lerr := recovery.Load(common)
	if lerr != nil {
		t.Fatal(lerr)
	}
	if len(f.Ops) != 0 {
		t.Fatalf("preparing record left behind: %+v", f.Ops)
	}
}

func TestUninitializedSubmoduleDoesNotBlockTrackedStash(t *testing.T) {
	m := setupRecoveryRepo(t)
	m.Outputs["version"] = "git version 2.54.0"
	m.Outputs["symbolic-ref --quiet HEAD"] = "refs/heads/main"
	m.Outputs["submodule status"] = "-deadbeef vendor/lib"
	m.FailOn["diff-index --quiet HEAD"] = fmt.Errorf("dirty")
	restoreID := recovery.NewID
	recovery.NewID = func() (string, error) { return "abc", nil }
	t.Cleanup(func() { recovery.NewID = restoreID })
	m.FailOn["stash push --message eg work.sync auto-stash abc"] = fmt.Errorf("index.lock")

	err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		t.Fatal("body ran")
		return nil
	})
	if err == nil || strings.Contains(err.Error(), "submodule") || !hasStashPush(m) {
		t.Fatalf("err=%v calls=%v", err, m.Calls)
	}
}

func TestDirtySubmoduleRefusesBeforeStash(t *testing.T) {
	m := setupRecoveryRepo(t)
	m.Outputs["version"] = "git version 2.54.0"
	m.Outputs["symbolic-ref --quiet HEAD"] = "refs/heads/main"
	m.Outputs["submodule status"] = "+deadbeef vendor/lib"
	m.FailOn["diff-index --quiet HEAD"] = fmt.Errorf("dirty")

	err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		t.Fatal("body ran")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "dirty submodule") || hasStashPush(m) {
		t.Fatalf("err=%v calls=%v", err, m.Calls)
	}
}

func TestCleanTreeDuringBisectStillRuns(t *testing.T) {
	m := setupRecoveryRepo(t)
	gitDir := m.Outputs["rev-parse --git-dir"]
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bisect := filepath.Join(gitDir, "BISECT_LOG")
	if err := os.WriteFile(bisect, []byte("git bisect start\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Outputs["rev-parse --git-path BISECT_LOG"] = bisect
	calls := 0
	if err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		calls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || hasStashPush(m) {
		t.Fatalf("calls=%d git=%v", calls, m.Calls)
	}
}

func TestEmptyStashPushRunsBodyWithoutRecord(t *testing.T) {
	m := setupRecoveryRepo(t)
	m.Outputs["version"] = "git version 2.54.0"
	m.Outputs["symbolic-ref --quiet HEAD"] = "refs/heads/main"
	m.FailOn["diff-index --quiet HEAD"] = fmt.Errorf("dirty")
	common := m.Outputs["rev-parse --git-common-dir"]
	calls := 0
	if err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		calls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !hasStashPush(m) {
		t.Fatalf("calls=%d git=%v", calls, m.Calls)
	}
	f, err := recovery.Load(common)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Ops) != 0 {
		t.Fatalf("record left behind: %+v", f.Ops)
	}
}

func TestCleanCommandRunsWhileAnotherRecoveryIsPending(t *testing.T) {
	m := setupRecoveryRepo(t)
	gitDir := m.Outputs["rev-parse --git-dir"]
	common := m.Outputs["rev-parse --git-common-dir"]
	op := recovery.Op{
		ID: "abc", Command: "work.sync", Worktree: gitDir, Source: "main",
		Phase: recovery.PhaseRunning, StashOID: "deadbeef", Ref: recovery.RefName("abc"),
	}
	if err := recovery.Save(common, recovery.File{Ops: []recovery.Op{op}}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	if err := StashPipe(cmdid.ID{Command: "work", Action: "start"}, func() error {
		calls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("body ran %d times", calls)
	}
}

func TestDirtyCommandRefusesWhileAnotherRecoveryIsPending(t *testing.T) {
	m := setupRecoveryRepo(t)
	gitDir := m.Outputs["rev-parse --git-dir"]
	common := m.Outputs["rev-parse --git-common-dir"]
	op := recovery.Op{
		ID: "abc", Command: "work.sync", Worktree: gitDir, Source: "main",
		Phase: recovery.PhaseRunning, StashOID: "deadbeef", Ref: recovery.RefName("abc"),
	}
	if err := recovery.Save(common, recovery.File{Ops: []recovery.Op{op}}); err != nil {
		t.Fatal(err)
	}
	m.FailOn["diff-index --quiet HEAD"] = fmt.Errorf("dirty")
	err := StashPipe(cmdid.ID{Command: "work", Action: "start"}, func() error {
		t.Fatal("body ran")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "pending recovery") || hasStashPush(m) {
		t.Fatalf("err=%v calls=%v", err, m.Calls)
	}
}

func TestLegacyFieldDoesNotReplayBody(t *testing.T) {
	m := setupRecoveryRepo(t)
	id := cmdid.ID{Command: "work", Action: "accept"}
	msg := "eg work.accept auto-stash: WIP in 'feat' branch on 2020-01-01T00:00:00"
	if err := cmdmem.Set(id, cmdmem.FieldStash, msg); err != nil {
		t.Fatal(err)
	}
	calls := 0
	err := StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "will not run again") || !strings.Contains(err.Error(), "eg repo doctor") {
		t.Fatalf("err=%v", err)
	}
	if calls != 0 || hasApply(m) {
		t.Fatalf("calls=%d git=%v", calls, m.Calls)
	}
	if got, gerr := cmdmem.Get(id, cmdmem.FieldStash); gerr != nil || got != msg {
		t.Fatalf("field %q err=%v", got, gerr)
	}
}

func TestAppliedCleanupClearsFieldBeforeDrop(t *testing.T) {
	m := setupRecoveryRepo(t)
	gitDir := m.Outputs["rev-parse --git-dir"]
	common := m.Outputs["rev-parse --git-common-dir"]
	id := cmdid.ID{Command: "work", Action: "sync"}
	msg := "eg work.sync auto-stash: token"
	op := recovery.Op{
		ID: "abc", Command: id.String(), Worktree: gitDir, Source: "main",
		Phase: recovery.PhaseApplied, StashOID: "deadbeef", Ref: recovery.RefName("abc"),
		Message: msg,
	}
	if err := recovery.Save(common, recovery.File{Ops: []recovery.Op{op}}); err != nil {
		t.Fatal(err)
	}
	if err := cmdmem.Set(id, cmdmem.FieldStash, msg); err != nil {
		t.Fatal(err)
	}
	listKey := strings.Join([]string{"stash", "list", "--format=%H%x00%gd%x00%s"}, " ")
	m.Outputs[listKey] = strings.Join([]string{"deadbeef", "stash@{0}", "On main: " + msg}, "\x00")
	m.FailOn["stash drop stash@{0}"] = fmt.Errorf("busy")
	calls := 0
	err := StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("err=%v calls=%#v", err, m.Calls)
	}
	if calls != 0 {
		t.Fatalf("body ran %d times", calls)
	}
	if got, gerr := cmdmem.Get(id, cmdmem.FieldStash); gerr != nil || got != "" {
		t.Fatalf("field %q err=%v", got, gerr)
	}
	f, loadErr := recovery.Load(common)
	if loadErr != nil || len(f.Ops) != 1 || f.Ops[0].Phase != recovery.PhaseApplied {
		t.Fatalf("journal %+v err=%v", f.Ops, loadErr)
	}
}

func TestSuspendedPhaseDoesNotReplayBody(t *testing.T) {
	m := setupRecoveryRepo(t)
	gitDir := m.Outputs["rev-parse --git-dir"]
	common := m.Outputs["rev-parse --git-common-dir"]
	id := cmdid.ID{Command: "work", Action: "sync"}
	op := recovery.Op{
		ID: "abc", Command: id.String(), Worktree: gitDir, Source: "main",
		Phase: recovery.PhaseSuspended, StashOID: "deadbeef", Ref: recovery.RefName("abc"),
	}
	if err := recovery.Save(common, recovery.File{Ops: []recovery.Op{op}}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	err := StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "will not run again") {
		t.Fatalf("err=%v", err)
	}
	if calls != 0 || hasApply(m) {
		t.Fatalf("calls=%d git=%v", calls, m.Calls)
	}
}

func TestSuspendedMissingSourceDoesNotRerun(t *testing.T) {
	m := setupRecoveryRepo(t)
	gitDir := m.Outputs["rev-parse --git-dir"]
	common := m.Outputs["rev-parse --git-common-dir"]
	id := cmdid.ID{Command: "work", Action: "sync"}
	op := recovery.Op{
		ID: "abc", Command: id.String(), Worktree: gitDir, Source: "main",
		Phase: recovery.PhaseSuspended, StashOID: "deadbeef", Ref: recovery.RefName("abc"),
	}
	if err := recovery.Save(common, recovery.File{Ops: []recovery.Op{op}}); err != nil {
		t.Fatal(err)
	}
	m.FailOn["rev-parse --verify --quiet --abbrev-ref --branches=refs/heads main"] = fmt.Errorf("missing")
	calls := 0
	err := StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "no longer exists") {
		t.Fatalf("err=%v", err)
	}
	if calls != 0 {
		t.Fatalf("body ran %d times", calls)
	}
}

func TestRunningPhaseDoesNotReplayBody(t *testing.T) {
	m := setupRecoveryRepo(t)
	gitDir := m.Outputs["rev-parse --git-dir"]
	common := m.Outputs["rev-parse --git-common-dir"]
	id := cmdid.ID{Command: "work", Action: "accept"}
	op := recovery.Op{
		ID: "abc", Command: id.String(), Worktree: gitDir, Source: "feat",
		Phase: recovery.PhaseRunning, StashOID: "deadbeef", Ref: recovery.RefName("abc"),
	}
	if err := recovery.Save(common, recovery.File{Ops: []recovery.Op{op}}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	err := StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "will not run again") {
		t.Fatalf("err=%v", err)
	}
	if calls != 0 {
		t.Fatalf("body ran %d times", calls)
	}
}

func TestPreservedFailureDoesNotApplyTwice(t *testing.T) {
	m := setupRecoveryRepo(t)
	gitDir := m.Outputs["rev-parse --git-dir"]
	common := m.Outputs["rev-parse --git-common-dir"]
	id := cmdid.ID{Command: "work", Action: "sync"}
	op := recovery.Op{
		ID: "abc", Command: id.String(), Worktree: gitDir, Source: "main",
		Phase: recovery.PhasePreserved, StashOID: "deadbeef", Ref: recovery.RefName("abc"),
	}
	if err := recovery.Save(common, recovery.File{Ops: []recovery.Op{op}}); err != nil {
		t.Fatal(err)
	}
	m.FailOn["stash apply --index deadbeef"] = fmt.Errorf("conflict")
	calls := 0
	err := StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("err=%v", err)
	}
	if calls != 1 || applyCount(m) != 1 {
		t.Fatalf("calls=%d applies=%d git=%v", calls, applyCount(m), m.Calls)
	}
	err = StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "will not be applied again") {
		t.Fatalf("second err=%v", err)
	}
	if calls != 1 || applyCount(m) != 1 {
		t.Fatalf("replayed calls=%d applies=%d", calls, applyCount(m))
	}
}

func setupRecoveryRepo(t *testing.T) *git.MemoryRunner {
	t.Helper()
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if resolved, err := filepath.EvalSymlinks(gitDir); err == nil {
		gitDir = resolved
	}
	m := git.NewMemoryRunner()
	m.Repo.CurrentBranch = "main"
	m.Outputs["rev-parse --git-dir"] = gitDir
	m.Outputs["rev-parse --git-common-dir"] = gitDir
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	return m
}

func hasStashPush(m *git.MemoryRunner) bool {
	for _, c := range m.Calls {
		if len(c.Args) >= 2 && c.Args[0] == "stash" && c.Args[1] == "push" {
			return true
		}
	}
	return false
}

func hasApply(m *git.MemoryRunner) bool {
	return applyCount(m) > 0
}

func applyCount(m *git.MemoryRunner) int {
	n := 0
	for _, c := range m.Calls {
		if len(c.Args) >= 2 && c.Args[0] == "stash" && c.Args[1] == "apply" {
			n++
		}
	}
	return n
}
