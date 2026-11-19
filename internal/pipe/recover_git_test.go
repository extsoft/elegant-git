package pipe

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/git"
	cmdmem "github.com/extsoft/elegant-git/internal/memory/cmd"
	"github.com/extsoft/elegant-git/internal/recovery"
	"github.com/extsoft/elegant-git/internal/text"
)

func TestHostGitMeetsRecoveryFloor(t *testing.T) {
	git.Use(git.RealRunner{})
	out, err := exec.Command(git.Executable(), "version").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if err := recovery.CheckFloor(); err != nil {
		t.Fatalf("host %s does not meet recovery floor %s: %v", strings.TrimSpace(string(out)), recovery.MinGitVersion, err)
	}
	t.Logf("recovery floor %s; host %s; os supported", recovery.MinGitVersion, strings.TrimSpace(string(out)))
}

func TestRealGitRestoresTrackedIndexSplit(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "staged\n")
	r.git("add", "f")
	r.write("f", "unstaged\n")
	id := cmdid.ID{Command: "work", Action: "sync"}
	if err := StashPipe(id, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	cached := r.git("diff", "--cached", "--", "f")
	unstaged := r.git("diff", "--", "f")
	if !strings.Contains(cached, "+staged") || !strings.Contains(unstaged, "+unstaged") {
		t.Fatalf("cached %q unstaged %q", cached, unstaged)
	}
	if list := r.git("stash", "list"); list != "" {
		t.Fatalf("temporary stash kept %q", list)
	}
	if refs := r.git("for-each-ref", recovery.RefPrefix); refs != "" {
		t.Fatalf("refs %q", refs)
	}
	if ignored := r.read("ignored"); ignored != "keep\n" {
		t.Fatalf("ignored %q", ignored)
	}
}

func TestRealGitSuspendedFailureDoesNotRerunOrRestore(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "dirty\n")
	id := cmdid.ID{Command: "work", Action: "accept"}
	calls := 0
	err := StashPipe(id, func() error {
		calls++
		return errors.New("publish failed")
	})
	if err == nil || !strings.Contains(err.Error(), "publish failed") {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	stash := r.git("stash", "list")
	if !strings.Contains(stash, "auto-stash") {
		t.Fatalf("stash %q", stash)
	}
	err = StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "will not run again") {
		t.Fatalf("suspended retry %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	if r.git("stash", "list") != stash {
		t.Fatalf("stash changed %q", r.git("stash", "list"))
	}
	if r.read("f") == "dirty\n" {
		t.Fatal("backup was applied")
	}
}

func TestRealGitSuspendedRebaseContinuationRunsBody(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "main\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "main2")
	r.git("checkout", "-q", "-b", "feat", "HEAD~1")
	r.write("f", "feat\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "feat")
	r.write("f", "dirty\n")
	id := cmdid.ID{Command: "work", Action: "sync"}
	calls := 0
	if err := StashPipe(id, func() error {
		calls++
		return r.gitErr("rebase", "main")
	}); err == nil {
		t.Fatal("expected rebase conflict")
	}
	if calls != 1 || r.git("stash", "list") == "" {
		t.Fatalf("calls=%d stash=%q", calls, r.git("stash", "list"))
	}
	stash := r.git("stash", "list")
	err := StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "rebase") {
		t.Fatalf("continuation %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
	if r.git("stash", "list") != stash {
		t.Fatalf("stash changed %q", r.git("stash", "list"))
	}
	if r.read("f") == "dirty\n" {
		t.Fatal("backup applied during rebase")
	}
	op := r.oneRecoveryOp(t)
	if op.RebaseHead != "refs/heads/feat" || op.RebaseOnto == "" || op.RebaseOrig == "" || op.Phase != recovery.PhaseSuspended {
		t.Fatalf("owned rebase %+v", op)
	}
}

func TestRealGitExternalRebaseAbortDoesNotReplay(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "main\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "main2")
	r.git("checkout", "-q", "-b", "feat", "HEAD~1")
	r.write("f", "feat\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "feat")
	r.write("f", "dirty\n")
	id := cmdid.ID{Command: "work", Action: "sync"}
	if err := StashPipe(id, func() error { return r.gitErr("rebase", "main") }); err == nil {
		t.Fatal("expected rebase conflict")
	}
	stash := r.git("stash", "list")
	r.git("rebase", "--abort")
	calls := 0
	err := StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "will not run again") {
		t.Fatalf("err=%v", err)
	}
	if calls != 0 || r.git("stash", "list") != stash || r.read("f") == "dirty\n" {
		t.Fatalf("calls=%d stash=%q file=%q", calls, r.git("stash", "list"), r.read("f"))
	}
}

func TestRealGitExternalRebaseContinueDoesNotReplay(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "main\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "main2")
	r.git("checkout", "-q", "-b", "feat", "HEAD~1")
	r.write("f", "feat\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "feat")
	r.write("f", "dirty\n")
	id := cmdid.ID{Command: "work", Action: "sync"}
	if err := StashPipe(id, func() error { return r.gitErr("rebase", "main") }); err == nil {
		t.Fatal("expected rebase conflict")
	}
	stash := r.git("stash", "list")
	r.write("f", "resolved\n")
	r.git("add", "f")
	if err := r.gitErr("-c", "core.editor=true", "rebase", "--continue"); err != nil {
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
	if calls != 0 || r.git("stash", "list") != stash || r.read("f") == "dirty\n" {
		t.Fatalf("calls=%d stash=%q file=%q", calls, r.git("stash", "list"), r.read("f"))
	}
}

func TestRealGitAbortThenUnrelatedRebaseIsNotContinuation(t *testing.T) {
	r, id := divergentFeat(t)
	if err := StashPipe(id, func() error { return r.gitErr("rebase", "main") }); err == nil {
		t.Fatal("expected rebase conflict")
	}
	r.git("rebase", "--abort")
	r.git("checkout", "-q", "-b", "unrelated", "main~1")
	r.write("f", "unrelated\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "unrelated")
	if err := r.gitErr("rebase", "main"); err == nil {
		t.Fatal("expected unrelated conflict")
	}
	hold := r.recoveryHold(t)
	calls := 0
	err := StashPipe(id, func() error {
		calls++
		return r.gitErr("rebase", "--continue")
	})
	if err == nil || !strings.Contains(err.Error(), "not owned by this backup") {
		t.Fatalf("err=%v", err)
	}
	if calls != 0 || r.recoveryHold(t) != hold || r.read("f") == "dirty\n" {
		t.Fatalf("calls=%d file=%q", calls, r.read("f"))
	}
	if got := rebaseHeadName(t, r); got != "refs/heads/unrelated" {
		t.Fatalf("rebase head %q", got)
	}
}

func TestRealGitReplacedRebaseIsNotContinuation(t *testing.T) {
	r, id := divergentFeat(t)
	if err := StashPipe(id, func() error { return r.gitErr("rebase", "main") }); err == nil {
		t.Fatal("expected rebase conflict")
	}
	r.git("rebase", "--abort")
	r.git("checkout", "-q", "main")
	r.write("g", "moved\n")
	r.git("add", "g")
	r.git("commit", "-q", "-m", "move main")
	r.git("checkout", "-q", "feat")
	if err := r.gitErr("rebase", "main"); err == nil {
		t.Fatal("expected replaced conflict")
	}
	hold := r.recoveryHold(t)
	calls := 0
	err := StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "not owned by this backup") {
		t.Fatalf("err=%v", err)
	}
	if calls != 0 || r.recoveryHold(t) != hold || r.read("f") == "dirty\n" {
		t.Fatalf("calls=%d file=%q", calls, r.read("f"))
	}
}

func TestRealGitUnownedSuspendedRebaseDoesNotContinue(t *testing.T) {
	r, id := divergentFeat(t)
	if err := StashPipe(id, func() error { return r.gitErr("rebase", "main") }); err == nil {
		t.Fatal("expected rebase conflict")
	}
	common := r.commonDir(t)
	file, err := recovery.Load(common)
	if err != nil {
		t.Fatal(err)
	}
	file.Ops[0].RebaseHead = ""
	file.Ops[0].RebaseOnto = ""
	file.Ops[0].RebaseOrig = ""
	if err := recovery.Save(common, file); err != nil {
		t.Fatal(err)
	}
	hold := r.recoveryHold(t)
	calls := 0
	err = StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "not owned by this backup") {
		t.Fatalf("err=%v", err)
	}
	if calls != 0 || r.recoveryHold(t) != hold || r.read("f") == "dirty\n" {
		t.Fatalf("calls=%d file=%q", calls, r.read("f"))
	}
	if r.oneRecoveryOp(t).Phase != recovery.PhaseSuspended {
		t.Fatal("journal was cleared")
	}
}

func TestRealGitAcceptHelperRebaseContinuesOnlyWhenOwned(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "main\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "main2")
	r.git("checkout", "-q", "-b", "__eg", "HEAD~1")
	r.write("f", "helper\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "helper")
	r.write("f", "dirty\n")
	id := cmdid.ID{Command: "work", Action: "accept"}
	calls := 0
	if err := StashPipe(id, func() error {
		calls++
		return r.gitErr("rebase", "main")
	}); err == nil {
		t.Fatal("expected helper conflict")
	}
	op := r.oneRecoveryOp(t)
	if op.RebaseHead != "refs/heads/__eg" {
		t.Fatalf("head %q", op.RebaseHead)
	}
	err := StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "rebase") || calls != 2 {
		t.Fatalf("continuation calls=%d err=%v", calls, err)
	}
	r.git("rebase", "--abort")
	r.git("checkout", "-q", "-b", "unrelated", "main~1")
	r.write("f", "unrelated\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "unrelated")
	if err := r.gitErr("rebase", "main"); err == nil {
		t.Fatal("expected unrelated conflict")
	}
	hold := r.recoveryHold(t)
	err = StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "not owned by this backup") {
		t.Fatalf("err=%v", err)
	}
	if calls != 2 || r.recoveryHold(t) != hold {
		t.Fatalf("calls=%d", calls)
	}
}

func TestRealGitDetachedHeadMoveKeepsBackup(t *testing.T) {
	r := newPipeRepo(t)
	r.git("checkout", "-q", "--detach")
	before := r.git("rev-parse", "HEAD")
	r.write("f", "dirty\n")
	id := cmdid.ID{Command: "work", Action: "accept"}
	if err := StashPipe(id, func() error {
		r.git("commit", "--allow-empty", "-q", "-m", "move")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if r.git("rev-parse", "HEAD") == before {
		t.Fatal("HEAD did not move")
	}
	if r.read("f") == "dirty\n" || r.git("stash", "list") == "" {
		t.Fatalf("file %q stash %q", r.read("f"), r.git("stash", "list"))
	}
	calls := 0
	err := StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "no longer detached") {
		t.Fatalf("err=%v", err)
	}
	if calls != 0 || r.read("f") == "dirty\n" {
		t.Fatalf("calls=%d file %q", calls, r.read("f"))
	}
}

func TestRealGitDetachedCheckoutToBranchKeepsBackup(t *testing.T) {
	r := newPipeRepo(t)
	r.git("checkout", "-q", "--detach")
	r.write("f", "dirty\n")
	id := cmdid.ID{Command: "work", Action: "accept"}
	if err := StashPipe(id, func() error {
		r.git("checkout", "-q", "main")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if r.git("rev-parse", "--abbrev-ref", "HEAD") != "main" {
		t.Fatal(r.git("rev-parse", "--abbrev-ref", "HEAD"))
	}
	if r.read("f") == "dirty\n" || r.git("stash", "list") == "" {
		t.Fatalf("file %q stash %q", r.read("f"), r.git("stash", "list"))
	}
}

func TestRealGitNestedRepoIsLeftUntouched(t *testing.T) {
	r := newPipeRepo(t)
	nested := filepath.Join(r.dir, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	sub := &pipeRepo{t: t, dir: nested}
	sub.git("init", "-q", "-b", "main")
	sub.git("config", "user.email", "t@example.com")
	sub.git("config", "user.name", "Test")
	sub.commit("base", "keep", "nested\n")
	sub.write("keep", "dirty-nested\n")
	head, err := os.ReadFile(filepath.Join(nested, ".git", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	r.write("f", "parent-dirty\n")
	if err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if sub.read("keep") != "dirty-nested\n" {
		t.Fatalf("nested %q", sub.read("keep"))
	}
	got, err := os.ReadFile(filepath.Join(nested, ".git", "HEAD"))
	if err != nil || string(got) != string(head) {
		t.Fatalf("HEAD %q err=%v", got, err)
	}
	if r.read("f") != "parent-dirty\n" {
		t.Fatalf("parent %q", r.read("f"))
	}
}

func TestRealGitRebaseKeepsStagedResolution(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "main\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "main2")
	r.git("checkout", "-q", "-b", "feat", "HEAD~1")
	r.write("f", "feat\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "feat")
	if err := r.gitErr("rebase", "main"); err == nil {
		t.Fatal("expected rebase conflict")
	}
	r.write("f", "resolved\n")
	r.git("add", "f")
	cached := r.git("diff", "--cached", "--", "f")
	if err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if r.git("stash", "list") != "" {
		t.Fatalf("stashed resolution %s", r.git("stash", "list"))
	}
	if r.read("f") != "resolved\n" || r.git("diff", "--cached", "--", "f") != cached {
		t.Fatalf("file %q cached %q", r.read("f"), r.git("diff", "--cached", "--", "f"))
	}
}

func TestRealGitRunningPhaseDoesNotReplay(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "dirty\n")
	id := cmdid.ID{Command: "work", Action: "accept"}
	if err := StashPipe(id, func() error { return errors.New("publish failed") }); err == nil {
		t.Fatal("expected publish failed")
	}
	common := r.git("rev-parse", "--git-common-dir")
	if !filepath.IsAbs(common) {
		common = filepath.Join(r.dir, common)
	}
	file, err := recovery.Load(common)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Ops) != 1 {
		t.Fatalf("ops %+v", file.Ops)
	}
	file.Ops[0].Phase = recovery.PhaseRunning
	if err := recovery.Save(common, file); err != nil {
		t.Fatal(err)
	}
	calls := 0
	err = StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "will not run again") {
		t.Fatalf("err=%v", err)
	}
	if calls != 0 {
		t.Fatalf("replayed calls=%d", calls)
	}
	if r.read("f") == "dirty\n" {
		t.Fatal("backup was applied")
	}
}

func TestRealGitDirtySubmoduleSameCommitRunsWithoutRecord(t *testing.T) {
	r := newPipeRepo(t)
	subDir := t.TempDir()
	sub := &pipeRepo{t: t, dir: subDir}
	sub.git("init", "-q", "-b", "main")
	sub.git("config", "user.email", "t@example.com")
	sub.git("config", "user.name", "Test")
	sub.commit("base", "f", "base\n")
	r.git("-c", "protocol.file.allow=always", "submodule", "add", "-q", subDir, "s")
	r.git("commit", "-q", "-m", "add sub")
	r.write("s/f", "changed\n")
	calls := 0
	if err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		calls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	if list := r.git("stash", "list"); list != "" {
		t.Fatalf("stash %q", list)
	}
	if r.read("s/f") != "changed\n" {
		t.Fatalf("submodule %q", r.read("s/f"))
	}
	common := r.git("rev-parse", "--git-common-dir")
	if !filepath.IsAbs(common) {
		common = filepath.Join(r.dir, common)
	}
	file, err := recovery.Load(common)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Ops) != 0 {
		t.Fatalf("ops %+v", file.Ops)
	}
}

func TestRealGitLegacySubjectProtectsOnlyWhenRecorded(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "dirty\n")
	msg := "eg work.sync auto-stash: WIP in 'main' branch on 2026"
	r.git("stash", "push", "-q", "--message", msg)
	subject := strings.TrimSpace(r.git("stash", "list", "--format=%s"))
	if !recovery.IsRecoverySubject(subject) || !strings.HasPrefix(subject, "On ") {
		t.Fatalf("subject %q", subject)
	}
	oid := strings.TrimSpace(r.git("rev-parse", "stash@{0}"))
	common := r.git("rev-parse", "--git-common-dir")
	if !filepath.IsAbs(common) {
		common = filepath.Join(r.dir, common)
	}
	protected, err := recovery.Protects(common, oid, subject)
	if err != nil || protected {
		t.Fatalf("orphan protected=%v err=%v subject=%q", protected, err, subject)
	}
	gitDir := r.git("rev-parse", "--git-dir")
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(r.dir, gitDir)
	}
	if err := os.MkdirAll(filepath.Join(gitDir, "elegant-git"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"work.sync":{"stash":"` + msg + `"}}` + "\n")
	if err := os.WriteFile(filepath.Join(gitDir, "elegant-git", "commands.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	protected, err = recovery.Protects(common, oid, subject)
	if err != nil || !protected {
		t.Fatalf("recorded protected=%v err=%v", protected, err)
	}
}

func TestRealGitApplyConflictIsNotRetried(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "two\n")
	id := cmdid.ID{Command: "work", Action: "sync"}
	err := StashPipe(id, func() error {
		r.write("f", "three\n")
		r.git("add", "f")
		r.git("commit", "-q", "-m", "other")
		return nil
	})
	if err == nil {
		t.Fatal("expected apply conflict")
	}
	stash := r.git("stash", "list")
	if stash == "" {
		t.Fatal("backup dropped")
	}
	calls := 0
	err = StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "will not be applied again") {
		t.Fatalf("second %v", err)
	}
	if calls != 0 {
		t.Fatal("body ran again")
	}
	if r.git("stash", "list") != stash {
		t.Fatalf("stash changed %q", r.git("stash", "list"))
	}
}

func TestRealGitMissingSourceKeepsBackupOffTheOtherBranch(t *testing.T) {
	r := newPipeRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.write("f", "feat-dirty\n")
	id := cmdid.ID{Command: "work", Action: "accept"}
	if err := StashPipe(id, func() error {
		r.git("checkout", "-q", "main")
		r.git("branch", "-D", "feat")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if r.read("f") == "feat-dirty\n" {
		t.Fatal("changes landed on main")
	}
	if r.git("stash", "list") == "" {
		t.Fatal("backup lost")
	}
	if err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if r.read("f") == "feat-dirty\n" {
		t.Fatal("clean command applied the backup")
	}
	r.write("f", "more\n")
	err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		t.Fatal("dirty command prepared")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "pending recovery") {
		t.Fatalf("other command %v", err)
	}
	if r.read("f") == "feat-dirty\n" {
		t.Fatal("later command applied the backup")
	}
}

func TestRealGitIntentToAddRefusesBeforeStash(t *testing.T) {
	r := newPipeRepo(t)
	r.write("new.txt", "new\n")
	r.git("add", "-N", "new.txt")
	err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		t.Fatal("body ran")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "intent-to-add") {
		t.Fatalf("err=%v", err)
	}
	if r.git("stash", "list") != "" {
		t.Fatalf("stashed %s", r.git("stash", "list"))
	}
}

func TestRealGitMergeRefusesBeforeStash(t *testing.T) {
	r := newPipeRepo(t)
	r.git("checkout", "-q", "-b", "other")
	r.write("f", "other\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "other")
	r.git("checkout", "-q", "main")
	r.write("f", "main\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "main2")
	if err := r.gitErr("merge", "other"); err == nil {
		t.Fatal("expected merge conflict")
	}
	err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		t.Fatal("body ran")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "merge") {
		t.Fatalf("err=%v", err)
	}
	if r.git("stash", "list") != "" {
		t.Fatalf("stashed %s", r.git("stash", "list"))
	}
}

func TestRealGitRebaseIsNotStashed(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "main\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "main2")
	r.git("checkout", "-q", "-b", "feat", "HEAD~1")
	r.write("f", "feat\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "feat")
	if err := r.gitErr("rebase", "main"); err == nil {
		t.Fatal("expected rebase conflict")
	}
	before := r.read("f")
	if err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if r.git("stash", "list") != "" {
		t.Fatalf("stashed a rebase: %s", r.git("stash", "list"))
	}
	if r.read("f") != before {
		t.Fatalf("worktree changed %q", r.read("f"))
	}
}

func TestRealGitOtherWorktreeCannotPrepare(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "dirty\n")
	id := cmdid.ID{Command: "work", Action: "push"}
	if err := StashPipe(id, func() error { return errors.New("stop") }); err != nil && !strings.Contains(err.Error(), "stop") {
		t.Fatal(err)
	}
	other := t.TempDir()
	r.git("worktree", "add", "-q", "-b", "other", other, "main")
	t.Chdir(other)
	if err := StashPipe(cmdid.ID{Command: "work", Action: "start"}, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "f"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := StashPipe(cmdid.ID{Command: "work", Action: "start"}, func() error {
		t.Fatal("other worktree prepared")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "pending recovery") {
		t.Fatalf("err=%v", err)
	}
}

func TestRealGitRecoveryStashStaysProtectedWhenRenumbered(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "dirty\n")
	if err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error { return errors.New("stop") }); err == nil {
		t.Fatal("expected stop")
	}
	oid := strings.TrimSpace(r.git("rev-parse", "stash@{0}"))
	r.write("g", "other\n")
	r.git("add", "g")
	r.git("stash", "push", "-q", "--message", "ordinary")
	r.git("stash", "drop", "stash@{0}")
	subject := r.git("stash", "list", "--format=%s")
	common := r.git("rev-parse", "--git-common-dir")
	if !filepath.IsAbs(common) {
		common = filepath.Join(r.dir, common)
	}
	protected, err := recovery.Protects(common, oid, subject)
	if err != nil {
		t.Fatal(err)
	}
	if !protected {
		t.Fatalf("oid %s subject %q not protected", oid, subject)
	}
	if !strings.Contains(r.git("stash", "list"), oid[:7]) && r.git("rev-parse", "stash@{0}") != oid && r.git("rev-parse", "stash@{1}") != oid {
		left := r.git("stash", "list", "--format=%H %s")
		if !strings.Contains(left, oid) {
			t.Fatalf("backup missing after renumber\n%s", left)
		}
	}
}

func TestRealGitCleanupInsertionLeavesUnrelatedStash(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "staged\n")
	r.git("add", "f")
	r.write("f", "unstaged\n")
	var owned, inserted string
	beforeRecoveryCleanup = func() {
		owned = r.git("rev-parse", "stash@{0}")
		r.write("g", "external\n")
		r.git("add", "g")
		r.git("stash", "push", "-q", "--message", "unrelated-external", "--", "g")
		inserted = r.git("rev-parse", "stash@{0}")
	}
	t.Cleanup(func() { beforeRecoveryCleanup = nil })
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	calls := 0
	if err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		calls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	cached := r.git("diff", "--cached", "--", "f")
	unstaged := r.git("diff", "--", "f")
	if !strings.Contains(cached, "+staged") || !strings.Contains(unstaged, "+unstaged") {
		t.Fatalf("cached %q unstaged %q", cached, unstaged)
	}
	if r.hasStash(owned) || !r.hasStash(inserted) || !strings.Contains(r.patch(inserted), "external") {
		t.Fatalf("stashes\n%s", r.git("stash", "list", "--format=%H %s"))
	}
	if refs := r.git("for-each-ref", recovery.RefPrefix); refs != "" {
		t.Fatalf("refs %q", refs)
	}
	if ops := r.recoveryOps(t); len(ops) != 0 {
		t.Fatalf("ops %+v", ops)
	}
	if strings.Contains(buf.String(), "did not drop") {
		t.Fatalf("output %q", buf.String())
	}
}

func TestRealGitCleanupRemovalLeavesReplacement(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "dirty\n")
	var replaced string
	beforeRecoveryCleanup = func() {
		r.git("stash", "drop", "stash@{0}")
		r.write("g", "replaced\n")
		r.git("add", "g")
		r.git("stash", "push", "-q", "--message", "replaced", "--", "g")
		replaced = r.git("rev-parse", "stash@{0}")
	}
	t.Cleanup(func() { beforeRecoveryCleanup = nil })
	calls := 0
	if err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		calls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || r.read("f") != "dirty\n" || !r.hasStash(replaced) || !strings.Contains(r.patch(replaced), "replaced") {
		t.Fatalf("calls=%d file=%q stashes\n%s", calls, r.read("f"), r.git("stash", "list", "--format=%H %s"))
	}
	if len(r.recoveryOps(t)) != 0 || r.git("for-each-ref", recovery.RefPrefix) != "" {
		t.Fatal("cleanup left the record")
	}
}

func TestRealGitCleanupABALeavesUnselectedStash(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "dirty\n")
	var owned, inserted string
	beforeRecoveryCleanup = func() {
		owned = r.git("rev-parse", "stash@{0}")
		r.git("stash", "drop", "stash@{0}")
		r.write("g", "external\n")
		r.git("add", "g")
		r.git("stash", "push", "-q", "--message", "unrelated-external", "--", "g")
		inserted = r.git("rev-parse", "stash@{0}")
		r.git("stash", "store", "-m", "restored", owned)
	}
	t.Cleanup(func() { beforeRecoveryCleanup = nil })
	err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error { return nil })
	if err == nil || !strings.Contains(err.Error(), "changed identity") || !strings.Contains(err.Error(), "Changes were restored") || !strings.Contains(err.Error(), owned) {
		t.Fatalf("err=%v", err)
	}
	if r.read("f") != "dirty\n" || !r.hasStash(owned) || !r.hasStash(inserted) {
		t.Fatalf("file %q stashes\n%s", r.read("f"), r.git("stash", "list", "--format=%H %s"))
	}
	if !strings.Contains(r.patch(inserted), "external") || !strings.Contains(r.patch(owned), "dirty") {
		t.Fatalf("patches\n%s", r.git("stash", "list", "--format=%H %s"))
	}
	ops := r.recoveryOps(t)
	if len(ops) != 1 || ops[0].Phase != recovery.PhaseApplied {
		t.Fatalf("ops %+v", ops)
	}
	if refs := r.git("for-each-ref", recovery.RefPrefix); !strings.Contains(refs, owned) {
		t.Fatalf("refs %q", refs)
	}
}

func TestRealGitAppliedCleanupDoesNotApplyAgain(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "dirty\n")
	r.git("add", "f")
	r.git("stash", "push", "-q", "--message", "owned")
	oid := r.git("rev-parse", "stash@{0}")
	r.saveApplied(t, oid)
	calls := 0
	if err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		calls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 0 || r.read("f") != "base\n" || r.hasStash(oid) {
		t.Fatalf("calls=%d file=%q stashes\n%s", calls, r.read("f"), r.git("stash", "list", "--format=%H %s"))
	}
	if ops := r.recoveryOps(t); len(ops) != 0 || r.git("for-each-ref", recovery.RefPrefix) != "" {
		t.Fatalf("ops %+v", ops)
	}
}

func TestRealGitDuplicateStashIdentityIsNotDropped(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "dirty\n")
	r.git("add", "f")
	r.git("stash", "push", "-q", "--message", "owned")
	oid := r.git("rev-parse", "stash@{0}")
	r.write("g", "other\n")
	r.git("add", "g")
	r.git("stash", "push", "-q", "--message", "other")
	r.git("stash", "store", "-m", "duplicate", oid)
	r.saveApplied(t, oid)
	calls := 0
	if err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		calls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 0 || !r.hasStash(oid) || !strings.Contains(r.patch(oid), "dirty") {
		t.Fatalf("calls=%d stashes\n%s", calls, r.git("stash", "list", "--format=%H %s"))
	}
	subjects := r.git("stash", "list", "--format=%gs")
	if strings.Contains(subjects, "owned") || !strings.Contains(subjects, "duplicate") || !strings.Contains(subjects, "other") {
		t.Fatalf("subjects %q", subjects)
	}
	if ops := r.recoveryOps(t); len(ops) != 0 || r.git("for-each-ref", recovery.RefPrefix) != "" {
		t.Fatalf("ops %+v", ops)
	}
}

func TestRealGitIndistinguishableStashIdentityIsNotDropped(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "dirty\n")
	r.git("add", "f")
	r.git("stash", "push", "-q", "--message", "owned")
	oid := r.git("rev-parse", "stash@{0}")
	r.write("g", "other\n")
	r.git("add", "g")
	r.git("stash", "push", "-q", "--message", "other")
	r.git("stash", "store", "-m", "owned", oid)
	r.saveApplied(t, oid)
	calls := 0
	err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), oid) || !strings.Contains(err.Error(), "ambiguous") || !strings.Contains(err.Error(), "Changes were restored") {
		t.Fatalf("err=%v", err)
	}
	if calls != 0 || !r.hasStash(oid) || !strings.Contains(r.patch(oid), "dirty") {
		t.Fatalf("calls=%d stashes\n%s", calls, r.git("stash", "list", "--format=%H %s"))
	}
	if strings.Count(r.git("stash", "list", "--format=%H"), oid) != 2 {
		t.Fatalf("stashes\n%s", r.git("stash", "list", "--format=%H %s"))
	}
	ops := r.recoveryOps(t)
	if len(ops) != 1 || ops[0].Phase != recovery.PhaseApplied {
		t.Fatalf("ops %+v", ops)
	}
	if refs := r.git("for-each-ref", recovery.RefPrefix); !strings.Contains(refs, oid) {
		t.Fatalf("refs %q", refs)
	}
}

func TestRealGitRestoringCleanupLeavesInsertedStash(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "dirty\n")
	var owned, inserted string
	beforeRecoveryCleanup = func() {
		owned = r.git("rev-parse", "stash@{0}")
		r.write("g", "external\n")
		r.git("add", "g")
		r.git("stash", "push", "-q", "--message", "unrelated-external", "--", "g")
		inserted = r.git("rev-parse", "stash@{0}")
	}
	t.Cleanup(func() { beforeRecoveryCleanup = nil })
	calls := 0
	err := StashPipeRestoring(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		calls++
		return errors.New("body failed")
	})
	if err == nil || !strings.Contains(err.Error(), "body failed") || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
	if r.read("f") != "dirty\n" || r.hasStash(owned) || !r.hasStash(inserted) || !strings.Contains(r.patch(inserted), "external") {
		t.Fatalf("file %q stashes\n%s", r.read("f"), r.git("stash", "list", "--format=%H %s"))
	}
	if ops := r.recoveryOps(t); len(ops) != 0 {
		t.Fatalf("ops %+v", ops)
	}
}

func TestRealGitLegacyMissingAndAmbiguousDropNothing(t *testing.T) {
	r := newPipeRepo(t)
	keeper := r.stash("keeper", "keeper-body\n")
	id := cmdid.ID{Command: "work", Action: "accept"}
	if err := cmdmem.Set(id, cmdmem.FieldStash, "eg work.accept auto-stash: missing"); err != nil {
		t.Fatal(err)
	}
	err := StashPipe(id, func() error {
		t.Fatal("body ran")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "was not found") || !strings.Contains(err.Error(), "other stashes were not changed") {
		t.Fatalf("err=%v", err)
	}
	if !r.hasStash(keeper) || !strings.Contains(r.patch(keeper), "keeper-body") {
		t.Fatalf("stashes\n%s", r.git("stash", "list", "--format=%H %s"))
	}

	msg := "eg work.accept auto-stash: same"
	one := r.stash(msg, "one\n")
	two := r.stash(msg, "two\n")
	if err := cmdmem.Set(id, cmdmem.FieldStash, msg); err != nil {
		t.Fatal(err)
	}
	err = StashPipe(id, func() error {
		t.Fatal("body ran")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "ambiguous") || !strings.Contains(err.Error(), "other stashes were not changed") {
		t.Fatalf("err=%v", err)
	}
	if !r.hasStash(one) || !r.hasStash(two) || !r.hasStash(keeper) {
		t.Fatalf("stashes\n%s", r.git("stash", "list", "--format=%H %s"))
	}
	if !strings.Contains(r.patch(one), "one") || !strings.Contains(r.patch(two), "two") {
		t.Fatalf("patches\n%s", r.git("stash", "list", "--format=%H %s"))
	}
}

func TestRealGitPopLegacyDoesNotPopAfterLookup(t *testing.T) {
	r := newPipeRepo(t)
	msg := "eg work.switch auto-stash: WIP in 'main' branch on 2026"
	owned := r.stash(msg, "owned-body\n")
	id := cmdid.ID{Command: "work", Action: "switch"}
	if err := cmdmem.Set(id, cmdmem.FieldStash, msg); err != nil {
		t.Fatal(err)
	}
	var inserted string
	beforeRecoveryCleanup = func() {
		inserted = r.stash("unrelated-external", "external-body\n")
	}
	t.Cleanup(func() { beforeRecoveryCleanup = nil })
	err := popLegacy(id)
	if err == nil || !strings.Contains(err.Error(), "will not pop") {
		t.Fatalf("err=%v", err)
	}
	if !r.hasStash(owned) || !r.hasStash(inserted) || !strings.Contains(r.patch(owned), "owned-body") || !strings.Contains(r.patch(inserted), "external-body") {
		t.Fatalf("stashes\n%s", r.git("stash", "list", "--format=%H %s"))
	}
	if got, gerr := cmdmem.Get(id, cmdmem.FieldStash); gerr != nil || got != msg {
		t.Fatalf("field %q err=%v", got, gerr)
	}
}

func TestRealGitRebaseThenSameCommitBranchDoesNotRestore(t *testing.T) {
	r := newPipeRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.write("g", "feat\n")
	r.git("add", "g")
	r.git("commit", "-q", "-m", "feat")
	r.git("checkout", "-q", "main")
	r.write("h", "main\n")
	r.git("add", "h")
	r.git("commit", "-q", "-m", "main2")
	r.git("checkout", "-q", "feat")
	r.write("f", "staged\n")
	r.git("add", "f")
	r.write("f", "unstaged\n")
	id := cmdid.ID{Command: "work", Action: "sync"}
	calls := 0
	err := StashPipe(id, func() error {
		calls++
		r.git("rebase", "main")
		r.git("branch", "other", "HEAD")
		r.git("checkout", "-q", "other")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "HEAD is other, not feat") {
		t.Fatal(err)
	}
	if calls != 1 || r.git("rev-parse", "--abbrev-ref", "HEAD") != "other" || r.read("f") != "base\n" || r.read("g") != "feat\n" {
		t.Fatalf("calls=%d head=%s f=%q g=%q", calls, r.git("rev-parse", "--abbrev-ref", "HEAD"), r.read("f"), r.read("g"))
	}
	if r.git("diff", "--cached") != "" || r.git("diff") != "" {
		t.Fatalf("index %q work %q", r.git("diff", "--cached"), r.git("diff"))
	}
	op := r.oneRecoveryOp(t)
	if op.Phase != recovery.PhaseSuspended || op.StashOID == "" || !r.hasStash(op.StashOID) {
		t.Fatalf("op %+v", op)
	}
	if refs := r.git("for-each-ref", op.Ref); !strings.Contains(refs, op.StashOID) {
		t.Fatalf("ref %q", refs)
	}
	stash := r.git("stash", "list")
	err = StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "will not run again") || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	if r.read("f") != "base\n" || r.git("stash", "list") != stash || r.git("rev-parse", "--abbrev-ref", "HEAD") != "other" {
		t.Fatal("second run changed the repo")
	}
}

func TestRealGitMovedAndDetachedHeadDoesNotRestore(t *testing.T) {
	r := newPipeRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.write("g", "feat\n")
	r.git("add", "g")
	r.git("commit", "-q", "-m", "feat")
	r.write("f", "dirty\n")
	id := cmdid.ID{Command: "work", Action: "sync"}
	err := StashPipe(id, func() error {
		r.git("checkout", "-q", "main")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "HEAD is main, not feat") || r.read("f") == "dirty\n" {
		t.Fatalf("err=%v file %q", err, r.read("f"))
	}
	if op := r.oneRecoveryOp(t); op.Phase != recovery.PhaseSuspended || !r.hasStash(op.StashOID) {
		t.Fatalf("op %+v", op)
	}

	r = newPipeRepo(t)
	r.write("f", "dirty\n")
	err = StashPipe(id, func() error {
		r.git("checkout", "-q", "--detach")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "detached at") || !strings.Contains(err.Error(), "not main") || r.read("f") == "dirty\n" {
		t.Fatalf("err=%v file %q", err, r.read("f"))
	}
	if r.git("rev-parse", "--abbrev-ref", "HEAD") != "HEAD" {
		t.Fatal(r.git("rev-parse", "--abbrev-ref", "HEAD"))
	}
}

func TestRealGitRecordedDestinationRestores(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "staged\n")
	r.git("add", "f")
	r.write("f", "unstaged\n")
	id := cmdid.ID{Command: "work", Action: "start"}
	if err := StashPipe(id, func() error {
		r.git("checkout", "-q", "-b", "feat")
		return RememberRestoreBranch("feat")
	}); err != nil {
		t.Fatal(err)
	}
	if r.git("rev-parse", "--abbrev-ref", "HEAD") != "feat" {
		t.Fatal(r.git("rev-parse", "--abbrev-ref", "HEAD"))
	}
	cached := r.git("diff", "--cached", "--", "f")
	unstaged := r.git("diff", "--", "f")
	if !strings.Contains(cached, "+staged") || !strings.Contains(unstaged, "+unstaged") {
		t.Fatalf("cached %q unstaged %q", cached, unstaged)
	}
	if refs := r.git("for-each-ref", recovery.RefPrefix); refs != "" || len(r.recoveryOps(t)) != 0 {
		t.Fatalf("refs %q ops %d", refs, len(r.recoveryOps(t)))
	}
	if list := r.git("stash", "list"); list != "" {
		t.Fatalf("temporary stash kept %q", list)
	}
}

func TestRealGitSameBranchCommitMoveRestores(t *testing.T) {
	r := newPipeRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.write("f", "staged\n")
	r.git("add", "f")
	r.write("f", "unstaged\n")
	id := cmdid.ID{Command: "work", Action: "sync"}
	if err := StashPipe(id, func() error {
		r.git("commit", "--allow-empty", "-q", "-m", "move")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if r.git("rev-parse", "--abbrev-ref", "HEAD") != "feat" {
		t.Fatal(r.git("rev-parse", "--abbrev-ref", "HEAD"))
	}
	if !strings.Contains(r.git("diff", "--cached", "--", "f"), "+staged") || !strings.Contains(r.git("diff", "--", "f"), "+unstaged") {
		t.Fatalf("cached %q unstaged %q", r.git("diff", "--cached", "--", "f"), r.git("diff", "--", "f"))
	}
}

func TestRealGitRestoreOnErrorRefusesOtherBranch(t *testing.T) {
	r := newPipeRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.git("branch", "other")
	r.write("f", "dirty\n")
	id := cmdid.ID{Command: "work", Action: "switch"}
	err := StashPipeRestoring(id, func() error {
		r.git("checkout", "-q", "other")
		return errors.New("switch failed")
	})
	if err == nil || !strings.Contains(err.Error(), "switch failed") || !strings.Contains(err.Error(), "HEAD is other, not feat") {
		t.Fatal(err)
	}
	if r.read("f") == "dirty\n" || r.git("rev-parse", "--abbrev-ref", "HEAD") != "other" {
		t.Fatalf("file %q head %s", r.read("f"), r.git("rev-parse", "--abbrev-ref", "HEAD"))
	}
	op := r.oneRecoveryOp(t)
	if op.Phase != recovery.PhaseSuspended || !r.hasStash(op.StashOID) {
		t.Fatalf("op %+v", op)
	}
	if refs := r.git("for-each-ref", op.Ref); !strings.Contains(refs, op.StashOID) {
		t.Fatalf("ref %q", refs)
	}
}

func TestRealGitBranchSwitchBetweenCheckAndApplyDoesNotRestore(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "dirty\n")
	beforeRestoreApply = func() { r.git("checkout", "-q", "-b", "other") }
	t.Cleanup(func() { beforeRestoreApply = nil })
	id := cmdid.ID{Command: "work", Action: "sync"}
	calls := 0
	err := StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "HEAD is other, not main") || !strings.Contains(err.Error(), "will not be applied again") {
		t.Fatal(err)
	}
	if calls != 1 || r.read("f") == "dirty\n" || r.git("diff") != "" || r.git("diff", "--cached") != "" {
		t.Fatalf("calls=%d file %q", calls, r.read("f"))
	}
	op := r.oneRecoveryOp(t)
	if op.Phase != recovery.PhaseApplying || !r.hasStash(op.StashOID) {
		t.Fatalf("op %+v", op)
	}
	err = StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "will not be applied again") || calls != 1 || r.read("f") == "dirty\n" {
		t.Fatalf("calls=%d err=%v file %q", calls, err, r.read("f"))
	}
}

func TestRealGitBranchSwitchAfterApplyKeepsRecord(t *testing.T) {
	r := newPipeRepo(t)
	r.git("branch", "other")
	r.write("f", "dirty\n")
	afterRestoreApply = func() { r.git("checkout", "-q", "other") }
	t.Cleanup(func() { afterRestoreApply = nil })
	id := cmdid.ID{Command: "work", Action: "sync"}
	calls := 0
	err := StashPipe(id, func() error {
		calls++
		return nil
	})
	op := r.oneRecoveryOp(t)
	if err == nil || !strings.Contains(err.Error(), op.StashOID) || !strings.Contains(err.Error(), "landed on other, not main") || !strings.Contains(err.Error(), "eg repo doctor") || !strings.Contains(err.Error(), "will not apply it again") {
		t.Fatal(err)
	}
	if calls != 1 || r.git("rev-parse", "--abbrev-ref", "HEAD") != "other" || r.read("f") != "dirty\n" {
		t.Fatalf("calls=%d head=%s file %q", calls, r.git("rev-parse", "--abbrev-ref", "HEAD"), r.read("f"))
	}
	if op.Phase != recovery.PhaseApplying || !r.hasStash(op.StashOID) {
		t.Fatalf("op %+v", op)
	}
	if refs := r.git("for-each-ref", op.Ref); !strings.Contains(refs, op.StashOID) {
		t.Fatalf("ref %q", refs)
	}
	err = StashPipe(id, func() error {
		calls++
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "will not be applied again") || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	if again := r.oneRecoveryOp(t); again.Phase != recovery.PhaseApplying || again.StashOID != op.StashOID || r.read("f") != "dirty\n" {
		t.Fatalf("op %+v file %q", again, r.read("f"))
	}
}

func TestRealGitAppliedResumeOnOtherBranchDoesNotApply(t *testing.T) {
	r := newPipeRepo(t)
	oid := r.stash("owned", "stashed\n")
	r.git("checkout", "-q", "-b", "other")
	r.write("f", "other\n")
	r.saveApplied(t, oid)
	calls := 0
	id := cmdid.ID{Command: "work", Action: "sync"}
	if err := StashPipe(id, func() error {
		calls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 0 || r.read("f") != "other\n" || r.git("rev-parse", "--abbrev-ref", "HEAD") != "other" || r.hasStash(oid) {
		t.Fatalf("calls=%d file %q stash %v", calls, r.read("f"), r.hasStash(oid))
	}
	if _, err := os.Stat(filepath.Join(r.dir, "extra")); !os.IsNotExist(err) {
		t.Fatal("applied onto other")
	}
	if len(r.recoveryOps(t)) != 0 || r.git("for-each-ref", recovery.RefPrefix) != "" {
		t.Fatal("cleanup left the record")
	}
}

func TestRealGitOtherWorktreeDoesNotRestore(t *testing.T) {
	r := newPipeRepo(t)
	other := t.TempDir()
	r.git("worktree", "add", "-q", "-b", "side", other, "HEAD")
	r.write("f", "dirty\n")
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	id := cmdid.ID{Command: "work", Action: "sync"}
	err = StashPipe(id, func() error {
		return os.Chdir(other)
	})
	if err == nil || !strings.Contains(err.Error(), "not the worktree") {
		t.Fatal(err)
	}
	body, readErr := os.ReadFile(filepath.Join(other, "f"))
	if readErr != nil || string(body) != "base\n" || r.read("f") == "dirty\n" {
		t.Fatalf("other %q primary %q", body, r.read("f"))
	}
	op := r.oneRecoveryOp(t)
	if op.Phase != recovery.PhaseSuspended || !r.hasStash(op.StashOID) {
		t.Fatalf("op %+v", op)
	}
}

func TestRealGitCarryDropsOwnedStashAndKeepsUnrelated(t *testing.T) {
	r := newPipeRepo(t)
	unrelated := r.stash("intentional", "keep-me\n")
	r.write("f", "staged\n")
	r.git("add", "f")
	r.write("f", "unstaged\n")
	r.write("free", "track-me\n")
	id := cmdid.ID{Command: "work", Action: "switch"}
	if _, err := StashPipeNavigation(id, false, "", func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	cached := r.git("diff", "--cached", "--", "f")
	unstaged := r.git("diff", "--", "f")
	if !strings.Contains(cached, "+staged") || !strings.Contains(unstaged, "+unstaged") || r.read("free") != "track-me\n" || r.read("ignored") != "keep\n" {
		t.Fatalf("cached %q unstaged %q free %q ignored %q", cached, unstaged, r.read("free"), r.read("ignored"))
	}
	if !r.hasStash(unrelated) || strings.Contains(r.git("stash", "list"), "auto-stash") {
		t.Fatalf("stashes\n%s", r.git("stash", "list", "--format=%H %gs"))
	}
	if !strings.Contains(r.patch(unrelated), "keep-me") {
		t.Fatal(r.patch(unrelated))
	}
	if refs := r.git("for-each-ref", recovery.RefPrefix); refs != "" || len(r.recoveryOps(t)) != 0 {
		t.Fatalf("refs %q ops %d", refs, len(r.recoveryOps(t)))
	}
	if got, err := cmdmem.Get(id, cmdmem.FieldStash); err != nil || got != "" {
		t.Fatalf("field %q err=%v", got, err)
	}
}

func TestRealGitHandoffKeepsLabeledStash(t *testing.T) {
	r := newPipeRepo(t)
	r.write("f", "dirty\n")
	kept, err := StashPipeNavigation(cmdid.ID{Command: "work", Action: "start"}, true, "kept label", func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if r.read("f") == "dirty\n" || !r.hasStash(kept.OID) || !strings.Contains(r.git("stash", "list"), "kept label") {
		t.Fatalf("file %q stash %q", r.read("f"), r.git("stash", "list"))
	}
	if len(r.recoveryOps(t)) != 0 || r.git("for-each-ref", recovery.RefPrefix) != "" {
		t.Fatal("handoff left recovery")
	}
	err = StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		r.write("g", "more\n")
		return errors.New("later")
	})
	if err == nil || !strings.Contains(err.Error(), "later") || !r.hasStash(kept.OID) || !strings.Contains(r.patch(kept.OID), "dirty") {
		t.Fatalf("err=%v stashes\n%s", err, r.git("stash", "list", "--format=%H %gs"))
	}
	if r.read("f") == "dirty\n" {
		t.Fatal("labeled stash was applied")
	}
}

func TestRealGitAppliedRetryWhenStashGone(t *testing.T) {
	r := newPipeRepo(t)
	oid := r.stash("owned", "dirty\n")
	r.saveApplied(t, oid)
	r.git("stash", "drop", "stash@{0}")
	calls := 0
	if err := StashPipe(cmdid.ID{Command: "work", Action: "sync"}, func() error {
		calls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 0 || len(r.recoveryOps(t)) != 0 || r.git("for-each-ref", recovery.RefPrefix) != "" {
		t.Fatalf("calls=%d ops=%d refs=%q", calls, len(r.recoveryOps(t)), r.git("for-each-ref", recovery.RefPrefix))
	}
}

func TestRealGitDropStashSelectsExactIdentity(t *testing.T) {
	r := newPipeRepo(t)
	raw := "keep\t\"quoted\""
	older := r.stash("older", "older\n")
	middle := r.stash(raw, "middle\n")
	newer := r.stash("newer", "newer\n")
	stored := ""
	for _, line := range strings.Split(r.git("stash", "list", "--format=%H %gs"), "\n") {
		if strings.HasPrefix(line, middle+" ") {
			stored = strings.TrimPrefix(line, middle+" ")
		}
	}
	msg := stored
	if rest, ok := strings.CutPrefix(stored, "On main: "); ok {
		msg = rest
	}
	if !strings.Contains(msg, "\"quoted\"") {
		t.Fatalf("stored %q", stored)
	}
	if msg != raw {
		if err := git.DropStash(middle, raw); err == nil || !strings.Contains(err.Error(), "changed identity") || !r.hasStash(middle) {
			t.Fatalf("err=%v", err)
		}
	}
	if err := git.DropStash(middle, msg); err != nil {
		t.Fatal(err)
	}
	if r.hasStash(middle) || !r.hasStash(older) || !r.hasStash(newer) {
		t.Fatalf("stashes\n%s", r.git("stash", "list", "--format=%H %gs"))
	}
	if err := git.DropStash(newer, "nope"); err == nil || !strings.Contains(err.Error(), "changed identity") || !r.hasStash(newer) {
		t.Fatalf("err=%v stashes\n%s", err, r.git("stash", "list", "--format=%H %gs"))
	}
	if err := git.DropStash(strings.Repeat("ab", 20), ""); err != nil {
		t.Fatal(err)
	}
	if err := git.DropStash(older, "older"); err != nil {
		t.Fatal(err)
	}
	if r.hasStash(older) || !r.hasStash(newer) {
		t.Fatalf("stashes\n%s", r.git("stash", "list", "--format=%H %gs"))
	}
}

func (r *pipeRepo) stash(message, body string) string {
	r.t.Helper()
	r.write("extra", body)
	r.git("add", "extra")
	r.git("stash", "push", "-q", "--message", message)
	return r.git("rev-parse", "stash@{0}")
}

func (r *pipeRepo) hasStash(oid string) bool {
	r.t.Helper()
	return strings.Contains(r.git("stash", "list", "--format=%H"), oid)
}

func (r *pipeRepo) patch(oid string) string {
	r.t.Helper()
	return r.git("stash", "show", "-p", oid)
}

func (r *pipeRepo) recoveryOps(t *testing.T) []recovery.Op {
	t.Helper()
	common := r.git("rev-parse", "--git-common-dir")
	if !filepath.IsAbs(common) {
		common = filepath.Join(r.dir, common)
	}
	file, err := recovery.Load(common)
	if err != nil {
		t.Fatal(err)
	}
	return file.Ops
}

func (r *pipeRepo) saveApplied(t *testing.T, oid string) {
	t.Helper()
	common := r.git("rev-parse", "--git-common-dir")
	if !filepath.IsAbs(common) {
		common = filepath.Join(r.dir, common)
	}
	gitDir := r.git("rev-parse", "--git-dir")
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(r.dir, gitDir)
	}
	op := recovery.Op{
		ID: "abc", Command: "work.sync", Worktree: gitDir, Source: "main",
		Phase: recovery.PhaseApplied, StashOID: oid, Ref: recovery.RefName("abc"),
		Message: "owned",
	}
	if err := recovery.Save(common, recovery.File{Ops: []recovery.Op{op}}); err != nil {
		t.Fatal(err)
	}
	r.git("update-ref", op.Ref, oid)
}

type pipeRepo struct {
	t   *testing.T
	dir string
}

func newPipeRepo(t *testing.T) *pipeRepo {
	t.Helper()
	unsetGitLocalEnv(t)
	dir := t.TempDir()
	r := &pipeRepo{t: t, dir: dir}
	r.git("init", "-q", "-b", "main")
	r.git("config", "user.email", "t@example.com")
	r.git("config", "user.name", "Test")
	r.commit("base", "f", "base\n")
	r.write(".gitignore", "ignored\n")
	r.git("add", ".gitignore")
	r.git("commit", "-q", "-m", "ignore")
	r.write("ignored", "keep\n")
	t.Chdir(dir)
	git.Use(git.RealRunner{})
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	return r
}

func (r *pipeRepo) git(args ...string) string {
	r.t.Helper()
	out, err := r.gitRaw(args...)
	if err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(out)
}

func (r *pipeRepo) gitErr(args ...string) error {
	r.t.Helper()
	_, err := r.gitRaw(args...)
	return err
}

func (r *pipeRepo) gitRaw(args ...string) (string, error) {
	cmd := exec.Command(git.Executable(), args...)
	cmd.Dir = r.dir
	cmd.Env = append(withoutGitLocalEnv(os.Environ()), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=t@example.com")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (r *pipeRepo) write(name, body string) {
	r.t.Helper()
	path := filepath.Join(r.dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *pipeRepo) read(name string) string {
	r.t.Helper()
	body, err := os.ReadFile(filepath.Join(r.dir, filepath.FromSlash(name)))
	if err != nil {
		r.t.Fatal(err)
	}
	return string(body)
}

func unsetGitLocalEnv(t *testing.T) {
	t.Helper()
	prev := map[string]string{}
	for _, key := range gitLocalEnvKeys {
		val, ok := os.LookupEnv(key)
		if !ok {
			continue
		}
		prev[key] = val
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for key, val := range prev {
			_ = os.Setenv(key, val)
		}
	})
}

func withoutGitLocalEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if gitLocalEnvKey(key) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func gitLocalEnvKey(key string) bool {
	for _, name := range gitLocalEnvKeys {
		if key == name {
			return true
		}
	}
	return false
}

var gitLocalEnvKeys = []string{
	"GIT_DIR",
	"GIT_WORK_TREE",
	"GIT_INDEX_FILE",
	"GIT_COMMON_DIR",
	"GIT_PREFIX",
	"GIT_OBJECT_DIRECTORY",
	"GIT_ALTERNATE_OBJECT_DIRECTORIES",
	"GIT_NAMESPACE",
}

func (r *pipeRepo) commonDir(t *testing.T) string {
	t.Helper()
	common := r.git("rev-parse", "--git-common-dir")
	if !filepath.IsAbs(common) {
		common = filepath.Join(r.dir, common)
	}
	return common
}

func (r *pipeRepo) oneRecoveryOp(t *testing.T) recovery.Op {
	t.Helper()
	file, err := recovery.Load(r.commonDir(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Ops) != 1 {
		t.Fatalf("ops %+v", file.Ops)
	}
	return file.Ops[0]
}

func (r *pipeRepo) recoveryHold(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(r.commonDir(t), "elegant-git", "recovery.json"))
	if err != nil {
		t.Fatal(err)
	}
	return r.git("stash", "list", "--format=%H %gd") + "\n" + r.git("for-each-ref", recovery.RefPrefix) + "\n" + string(data)
}

func rebaseHeadName(t *testing.T, r *pipeRepo) string {
	t.Helper()
	path := r.git("rev-parse", "--git-path", "rebase-merge/head-name")
	if !filepath.IsAbs(path) {
		path = filepath.Join(r.dir, path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(body))
}

func divergentFeat(t *testing.T) (*pipeRepo, cmdid.ID) {
	t.Helper()
	r := newPipeRepo(t)
	r.write("f", "main\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "main2")
	r.git("checkout", "-q", "-b", "feat", "HEAD~1")
	r.write("f", "feat\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "feat")
	r.write("f", "dirty\n")
	return r, cmdid.ID{Command: "work", Action: "sync"}
}

func (r *pipeRepo) commit(msg, name, body string) {
	r.t.Helper()
	r.write(name, body)
	r.git("add", name)
	r.git("commit", "-q", "-m", msg)
}
