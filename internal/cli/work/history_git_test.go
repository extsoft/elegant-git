package work

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/exitcode"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

func TestSyncDirtyNoninteractiveRefuses(t *testing.T) {
	r := newGitRepo(t)
	r.branch("feature", "main")
	r.write("f", "dirty\n")
	r.enter()
	r.setSource("feature", "main")
	cmd := newSyncCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	buf := captureText(t)
	err := syncRun(cmd, nil)
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "--dirty") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.git("rev-parse", "--abbrev-ref", "HEAD") != "feature" || r.readFile("f") != "dirty\n" || r.git("stash", "list") != "" {
		t.Fatalf("head %s file %q stash %q", r.git("rev-parse", "--abbrev-ref", "HEAD"), r.readFile("f"), r.git("stash", "list"))
	}
}

func TestSyncStashLeavesTheChanges(t *testing.T) {
	r := newGitRepo(t)
	r.branch("feature", "main")
	r.write("f", "dirty\n")
	r.enter()
	r.setSource("feature", "main")
	cmd := newSyncCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := cmd.Flags().Set("dirty", "stash"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("stash-name", "kept"); err != nil {
		t.Fatal(err)
	}
	buf := captureText(t)
	if err := syncRun(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if r.git("rev-parse", "--abbrev-ref", "HEAD") != "feature" || r.readFile("f") != "o\n" {
		t.Fatalf("head %s file %q", r.git("rev-parse", "--abbrev-ref", "HEAD"), r.readFile("f"))
	}
	if !strings.Contains(r.git("stash", "list"), "kept") || !strings.Contains(buf.String(), "kept") || strings.Contains(buf.String(), "will not be applied again") {
		t.Fatalf("stash %q out %q", r.git("stash", "list"), buf.String())
	}
}

func TestSyncBodyRefusalKeepsExitAndBackup(t *testing.T) {
	r := newGitRepo(t)
	r.branch("feature", "main")
	r.write("f", "dirty\n")
	r.enter()
	cmd := newSyncCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := cmd.Flags().Set("dirty", "carry"); err != nil {
		t.Fatal(err)
	}
	buf := captureText(t)
	err := PrepareHistory(History{
		Cmd:     cmd,
		ID:      syncID,
		Verb:    "work sync",
		Dest:    "the rebase",
		Resolve: func() (string, error) { return "HEAD", nil },
		Body: func() error {
			return workflowText("blocked on purpose")
		},
	})
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "blocked on purpose") || !strings.Contains(buf.String(), "The recovery backup was kept.") {
		t.Fatalf("out=%q err=%v", buf.String(), err)
	}
	if r.readFile("f") != "o\n" || r.git("stash", "list") == "" {
		t.Fatalf("file %q stash %q", r.readFile("f"), r.git("stash", "list"))
	}
}

func TestSaveAutosquashStashLeavesLeftovers(t *testing.T) {
	r := newGitRepo(t)
	r.branch("feature", "main")
	older := r.commit("older", "g", "older\n")
	r.commit("tip", "h", "tip\n")
	r.write("g", "fixed\n")
	r.git("add", "g")
	r.write("extra", "left\n")
	r.enter()
	t.Setenv("GIT_EDITOR", "true")
	t.Setenv("GIT_SEQUENCE_EDITOR", "true")
	cmd := newSaveCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := cmd.Flags().Set("dirty", "stash"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("stash-name", "kept"); err != nil {
		t.Fatal(err)
	}
	saved, err := applyFixup(cmd, older)
	if err != nil {
		t.Fatal(err)
	}
	if saved == "" || saved == r.git("rev-parse", "HEAD") {
		t.Fatalf("saved %s head %s", saved, r.git("rev-parse", "HEAD"))
	}
	if _, err := os.Stat(filepath.Join(r.dir, "extra")); !os.IsNotExist(err) {
		t.Fatalf("leftover returned: %v", err)
	}
	if r.readFile("g") != "fixed\n" || !strings.Contains(r.git("stash", "list"), "kept") {
		t.Fatalf("g %q stash %q", r.readFile("g"), r.git("stash", "list"))
	}
}

func TestSavedRefusalKeepsTheExitAndNamesTheCommit(t *testing.T) {
	buf := captureText(t)
	err := annotate(workflowText("cannot be carried"), "saved abc123; autosquash did not finish, and that commit was left in place")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "abc123") || !strings.Contains(buf.String(), "left in place") || !strings.Contains(buf.String(), "cannot be carried") {
		t.Fatalf("out=%q", buf.String())
	}
}

func TestSaveFixupRefusesNestedRepoBeforeCommit(t *testing.T) {
	r := newGitRepo(t)
	r.branch("feature", "main")
	older := r.commit("older", "g", "older\n")
	r.commit("tip", "h", "tip\n")
	r.write("g", "fixed\n")
	r.git("add", "g")
	nest := filepath.Join(r.dir, "nest")
	if err := os.MkdirAll(nest, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(git.Executable(), "init", "-q", "-b", "main")
	cmd.Dir = nest
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("nested init: %v\n%s", err, out)
	}
	before := r.git("rev-parse", "HEAD")
	r.enter()
	t.Setenv("GIT_EDITOR", "true")
	t.Setenv("GIT_SEQUENCE_EDITOR", "true")
	save := newSaveCommand()
	save.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := save.Flags().Set("dirty", "carry"); err != nil {
		t.Fatal(err)
	}
	buf := captureText(t)
	_, err := applyFixup(save, older)
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "nested repository") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.git("rev-parse", "HEAD") != before || strings.Contains(r.git("log", "--format=%s"), "fixup!") {
		t.Fatalf("head %s log %s", r.git("rev-parse", "HEAD"), r.git("log", "--format=%s"))
	}
}

func TestSyncCarryRestoresIndex(t *testing.T) {
	r := newGitRepo(t)
	r.branch("feature", "main")
	r.commit("c", "c", "c\n")
	r.write("f", "staged\n")
	r.git("add", "f")
	r.write("f", "unstaged\n")
	r.enter()
	r.setSource("feature", "main")
	cached := r.git("diff", "--cached")
	plain := r.git("diff")
	cmd := newSyncCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := cmd.Flags().Set("dirty", "carry"); err != nil {
		t.Fatal(err)
	}
	if err := syncRun(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if r.git("rev-parse", "--abbrev-ref", "HEAD") != "feature" {
		t.Fatalf("head %s", r.git("rev-parse", "--abbrev-ref", "HEAD"))
	}
	if r.git("diff", "--cached") != cached || r.git("diff") != plain {
		t.Fatalf("cached %q plain %q", r.git("diff", "--cached"), r.git("diff"))
	}
}

func TestAcceptRefusesForeignRebaseBeforeStash(t *testing.T) {
	r := newGitRepo(t)
	r.branch("feature", "main")
	r.write("f", "feature\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "feature")
	r.checkout("main")
	r.write("f", "main\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "main")
	r.checkout("feature")
	if err := r.gitErr("rebase", "main"); err == nil {
		t.Fatal("expected a conflict")
	}
	r.write("f", "resolved\n")
	r.git("add", "f")
	r.enter()
	var branch string
	spec := acceptSpec(&branch)
	cmd := newAcceptCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	buf := captureText(t)
	err := acceptRun(cmd, []string{"feature"}, spec)
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "rebase") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.git("stash", "list") != "" || r.readFile("f") != "resolved\n" {
		t.Fatalf("stash %q file %q", r.git("stash", "list"), r.readFile("f"))
	}
	if !strings.Contains(r.git("diff", "--cached"), "resolved") {
		t.Fatalf("resolution left the index: %s", r.git("diff", "--cached"))
	}
}

func TestPolishContinuesOnProtectedBranch(t *testing.T) {
	r := newGitRepo(t)
	r.branch("feature", "main")
	r.write("f", "feature\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "feature")
	r.checkout("main")
	r.write("f", "main\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "main")
	r.checkout("feature")
	if err := r.gitErr("rebase", "main"); err == nil {
		t.Fatal("expected a conflict")
	}
	r.write("f", "resolved\n")
	r.git("add", "f")
	r.enter()
	t.Setenv("GIT_EDITOR", "true")
	t.Setenv("GIT_SEQUENCE_EDITOR", "true")
	cmd := newPolishCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := polishRun(cmd); err != nil {
		t.Fatal(err)
	}
	if state.IsThereActiveRebase() {
		t.Fatal("rebase still active")
	}
	if r.readFile("f") != "resolved\n" || r.git("stash", "list") != "" {
		t.Fatalf("file %q stash %q", r.readFile("f"), r.git("stash", "list"))
	}
}

func TestSaveStashOnProtectedDoesNotCommit(t *testing.T) {
	r := newGitRepo(t)
	r.write("f", "staged\n")
	r.git("add", "f")
	r.write("f", "unstaged\n")
	cached := r.git("diff", "--cached")
	plain := r.git("diff")
	r.enter()
	before := r.git("rev-parse", "HEAD")
	cmd := newSaveCommand()
	if err := cmd.Flags().Set("dirty", "stash"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("stash-name", "kept"); err != nil {
		t.Fatal(err)
	}
	cmd.SetContext(prompt.WithPrompter(context.Background(), &sessionPrompter{strings: []string{"feature", "kept"}}))
	var target string
	buf := captureText(t)
	err := saveViaStartOnProtected(cmd, "main", nil, saveSpec(&target))
	assertWorkflow(t, err)
	if r.git("rev-parse", "--abbrev-ref", "HEAD") != "feature" || r.git("rev-parse", "HEAD") != before {
		t.Fatalf("head %s %s", r.git("rev-parse", "--abbrev-ref", "HEAD"), r.git("rev-parse", "HEAD"))
	}
	oid := r.git("rev-parse", "stash@{0}")
	out := buf.String()
	if r.readFile("f") != "o\n" || !strings.Contains(r.git("stash", "list"), "kept") || !strings.Contains(out, oid) || !strings.Contains(out, "did not commit") || !strings.Contains(out, "git stash apply --index "+oid) {
		t.Fatalf("file %q stash %q out %q", r.readFile("f"), r.git("stash", "list"), out)
	}
	r.git("stash", "apply", "--index", oid)
	if r.git("diff", "--cached") != cached || r.git("diff") != plain {
		t.Fatalf("cached %q plain %q", r.git("diff", "--cached"), r.git("diff"))
	}
}

func TestSaveDiscardOnProtectedDoesNotReset(t *testing.T) {
	r := newGitRepo(t)
	r.write("f", "dirty\n")
	r.enter()
	cmd := newSaveCommand()
	if err := cmd.Flags().Set("dirty", "discard"); err != nil {
		t.Fatal(err)
	}
	cmd.SetContext(prompt.WithPrompter(context.Background(), &sessionPrompter{strings: []string{"feature"}}))
	var target string
	buf := captureText(t)
	err := saveViaStartOnProtected(cmd, "main", nil, saveSpec(&target))
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "will not discard") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.git("rev-parse", "--abbrev-ref", "HEAD") != "main" || r.readFile("f") != "dirty\n" || r.git("stash", "list") != "" {
		t.Fatalf("head %s file %q stash %q", r.git("rev-parse", "--abbrev-ref", "HEAD"), r.readFile("f"), r.git("stash", "list"))
	}
	if got := r.gitAllow("rev-parse", "--verify", "--quiet", "refs/heads/feature"); got != "" {
		t.Fatalf("branch %s", got)
	}
}

func TestHistoryRefusesIgnoredCollisionBeforePrep(t *testing.T) {
	t.Run("sync carry", func(t *testing.T) {
		r, head := ignoredUpstreamRepo(t)
		r.enter()
		r.setSource("feature", "main")
		refuseIgnored(t, r, head, "b.txt", func() error {
			cmd := dirtyCmd(t, newSyncCommand(), "carry")
			return syncRun(cmd, nil)
		})
	})
	t.Run("sync discard from subdirectory", func(t *testing.T) {
		r, head := ignoredUpstreamRepo(t)
		r.enter()
		r.setSource("feature", "main")
		sub := filepath.Join(r.dir, "sub")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(sub); err != nil {
			t.Fatal(err)
		}
		refuseIgnored(t, r, head, "b.txt", func() error {
			cmd := dirtyCmd(t, newSyncCommand(), "discard")
			return syncRun(cmd, nil)
		})
	})
	t.Run("push", func(t *testing.T) {
		r, head := ignoredUpstreamRepo(t)
		r.enter()
		r.setSource("feature", "main")
		refuseIgnored(t, r, head, "b.txt", func() error {
			cmd := dirtyCmd(t, newPushCommand(), "carry")
			return pushRun(cmd, nil)
		})
	})
	t.Run("polish", func(t *testing.T) {
		r := newGitRepo(t)
		r.branch("feature", "main")
		r.commit("g", "g.txt", "g\n")
		if err := os.MkdirAll(filepath.Join(r.dir, "vendor"), 0o755); err != nil {
			t.Fatal(err)
		}
		r.write("vendor/lib.txt", "v\n")
		r.git("add", "-f", "vendor/lib.txt")
		r.git("commit", "-q", "-m", "track vendor")
		r.commit("h", "h.txt", "h\n")
		r.git("rm", "--cached", "-q", "vendor/lib.txt")
		r.git("commit", "-q", "-m", "untrack")
		r.write(".gitignore", "vendor/\n")
		r.git("add", ".gitignore")
		r.git("commit", "-q", "-m", "ignore")
		r.write("vendor/lib.txt", "mine ignored\n")
		r.write("f", "my tracked edit\n")
		head := r.git("rev-parse", "HEAD")
		r.enter()
		r.setSource("feature", "main")
		refuseIgnored(t, r, head, "f", func() error {
			cmd := dirtyCmd(t, newPolishCommand(), "carry")
			return polishRun(cmd)
		})
	})
	t.Run("accept", func(t *testing.T) {
		r, head := ignoredUpstreamRepo(t)
		r.enter()
		r.setSource("feature", "main")
		refuseIgnored(t, r, head, "b.txt", func() error {
			cmd := dirtyCmd(t, newAcceptCommand(), "carry")
			var branch string
			return acceptRun(cmd, []string{"feature"}, acceptSpec(&branch))
		})
	})
}

func TestSaveRejectsInvalidDirtyBeforeAnyEffect(t *testing.T) {
	r := newGitRepo(t)
	r.write("f", "dirty\n")
	head := r.git("rev-parse", "HEAD")
	r.enter()
	cmd := newSaveCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := cmd.Flags().Set("dirty", "z"); err != nil {
		t.Fatal(err)
	}
	var target string
	err := saveRun(cmd, nil, saveSpec(&target))
	if !cliruntime.IsUsageError(err) || !strings.Contains(err.Error(), "carry") {
		t.Fatalf("%v", err)
	}
	cmd = newSaveCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := cmd.Flags().Set("stash-name", "kept"); err != nil {
		t.Fatal(err)
	}
	err = saveRun(cmd, nil, saveSpec(&target))
	if !cliruntime.IsUsageError(err) || !strings.Contains(err.Error(), "--stash-name") {
		t.Fatalf("%v", err)
	}
	if r.git("rev-parse", "HEAD") != head || r.readFile("f") != "dirty\n" || r.git("stash", "list") != "" {
		t.Fatalf("head %s file %q stash %q", r.git("rev-parse", "HEAD"), r.readFile("f"), r.git("stash", "list"))
	}
}

func TestDiscardKeepsStagedAdditionsAndRenameTargets(t *testing.T) {
	r := newGitRepo(t)
	r.commit("d", "d.txt", "d\n")
	r.commit("e", "e.txt", "e\n")
	r.commit("renamed", "f.txt", "f\n")
	r.branch("other", "main")
	r.checkout("main")
	r.write("d.txt", "changed\n")
	r.git("add", "d.txt")
	r.git("rm", "-q", "e.txt")
	r.write("new.txt", "new\n")
	r.git("add", "new.txt")
	r.git("mv", "f.txt", "g.txt")
	r.write("stay.txt", "stay\n")
	r.enter()
	cmd := newSwitchCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := cmd.Flags().Set("dirty", "discard"); err != nil {
		t.Fatal(err)
	}
	if err := switchRun(cmd, []string{"other"}); err != nil {
		t.Fatal(err)
	}
	if r.git("rev-parse", "--abbrev-ref", "HEAD") != "other" {
		t.Fatalf("head %s", r.git("rev-parse", "--abbrev-ref", "HEAD"))
	}
	if r.readFile("d.txt") != "d\n" || r.readFile("e.txt") != "e\n" || r.readFile("f.txt") != "f\n" {
		t.Fatalf("d %q e %q f %q", r.readFile("d.txt"), r.readFile("e.txt"), r.readFile("f.txt"))
	}
	if r.readFile("new.txt") != "new\n" || r.readFile("g.txt") != "f\n" || r.readFile("stay.txt") != "stay\n" {
		t.Fatalf("new %q g %q stay %q", r.readFile("new.txt"), r.readFile("g.txt"), r.readFile("stay.txt"))
	}
	status := r.git("status", "--porcelain")
	for _, path := range []string{"new.txt", "g.txt", "stay.txt"} {
		if !strings.Contains(status, "?? "+path) {
			t.Fatalf("status %q missing %s", status, path)
		}
	}
	if r.gitAllow("ls-files", "--error-unmatch", "new.txt") != "" || r.gitAllow("ls-files", "--error-unmatch", "g.txt") != "" {
		t.Fatal("staged additions stayed in the index")
	}
}

func TestAcceptHelperContinueMerges(t *testing.T) {
	r := newGitRepo(t)
	r.branch("feature", "main")
	r.write("f", "feature\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "feature")
	r.checkout("main")
	r.write("f", "main\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "main")
	r.checkout("feature")
	r.write("g.txt", "dirty\n")
	r.enter()
	r.setSource("feature", "main")
	t.Setenv("GIT_EDITOR", "true")
	t.Setenv("GIT_SEQUENCE_EDITOR", "true")
	cmd := newAcceptCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := cmd.Flags().Set("dirty", "carry"); err != nil {
		t.Fatal(err)
	}
	var branch string
	if err := acceptRun(cmd, []string{"feature"}, acceptSpec(&branch)); err == nil || !state.IsThereActiveRebase() {
		t.Fatalf("err=%v rebase=%v", err, state.IsThereActiveRebase())
	}
	r.write("f", "resolved\n")
	r.git("add", "f")
	cmd = newAcceptCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := cmd.Flags().Set("dirty", "carry"); err != nil {
		t.Fatal(err)
	}
	if err := acceptRun(cmd, []string{"feature"}, acceptSpec(&branch)); err != nil {
		t.Fatal(err)
	}
	if state.IsThereActiveRebase() || r.git("rev-parse", "--abbrev-ref", "HEAD") != "main" || r.readFile("f") != "resolved\n" {
		t.Fatalf("rebase %v head %s f %q", state.IsThereActiveRebase(), r.git("rev-parse", "--abbrev-ref", "HEAD"), r.readFile("f"))
	}
	if r.gitAllow("rev-parse", "--verify", "--quiet", "refs/heads/__eg") != "" {
		t.Fatal("helper remains")
	}
}

func TestAcceptRefusesToRebuildHelper(t *testing.T) {
	r := newGitRepo(t)
	r.branch("feature", "main")
	r.commit("feat", "g.txt", "g\n")
	r.git("checkout", "-q", "-B", "__eg", "feature")
	extra := r.commit("extra", "h.txt", "extra\n")
	r.checkout("feature")
	r.enter()
	r.setSource("feature", "main")
	cmd := newAcceptCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	buf := captureText(t)
	var branch string
	err := acceptRun(cmd, []string{"feature"}, acceptSpec(&branch))
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "will not replace") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.git("rev-parse", "__eg") != extra || r.git("rev-parse", "--abbrev-ref", "HEAD") != "feature" {
		t.Fatalf("helper %s head %s", r.git("rev-parse", "__eg"), r.git("rev-parse", "--abbrev-ref", "HEAD"))
	}
}

func TestAcceptDeletesUpstreamBranch(t *testing.T) {
	r := newGitRepo(t)
	r.addOrigin("main")
	r.branch("feature", "main")
	r.commit("feat", "g.txt", "g\n")
	r.git("push", "-u", "origin", "feature")
	r.enter()
	r.setSource("feature", "main")
	cmd := newAcceptCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	buf := captureText(t)
	var branch string
	if err := acceptRun(cmd, []string{"feature"}, acceptSpec(&branch)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Deleted the remote branch 'feature'.") {
		t.Fatalf("out=%q", buf.String())
	}
	url := r.git("remote", "get-url", "origin")
	show := exec.Command(git.Executable(), "--git-dir", url, "show-ref", "--verify", "--quiet", "refs/heads/feature")
	if err := show.Run(); err == nil {
		t.Fatal("remote feature remains")
	}
	if r.gitAllow("rev-parse", "--verify", "--quiet", "refs/heads/feature") != "" || r.git("rev-parse", "--abbrev-ref", "HEAD") != "main" || r.readFile("g.txt") != "g\n" {
		t.Fatalf("head %s", r.git("rev-parse", "--abbrev-ref", "HEAD"))
	}
}

func TestAcceptDoesNotDeleteDefaultUpstream(t *testing.T) {
	r := newGitRepo(t)
	r.addOrigin("main")
	r.git("checkout", "-q", "-b", "feature", "origin/main")
	r.commit("feat", "g.txt", "g\n")
	r.enter()
	r.setSource("feature", "main")
	cmd := newAcceptCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	buf := captureText(t)
	var branch string
	if err := acceptRun(cmd, []string{"feature"}, acceptSpec(&branch)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "Deleted the remote branch") {
		t.Fatalf("out=%q", buf.String())
	}
	url := r.git("remote", "get-url", "origin")
	show := exec.Command(git.Executable(), "--git-dir", url, "show-ref", "--verify", "--quiet", "refs/heads/main")
	if err := show.Run(); err != nil {
		t.Fatal("origin/main was deleted")
	}
	if r.readFile("g.txt") != "g\n" || r.git("rev-parse", "--abbrev-ref", "HEAD") != "main" {
		t.Fatalf("head %s", r.git("rev-parse", "--abbrev-ref", "HEAD"))
	}
}

func TestAcceptContinueRejectsADifferentBranch(t *testing.T) {
	r := newGitRepo(t)
	r.branch("feature", "main")
	r.write("f", "feature\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "feature")
	r.checkout("main")
	r.branch("other", "main")
	r.commit("other", "h.txt", "other\n")
	r.checkout("main")
	r.write("f", "main\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "main")
	r.checkout("feature")
	r.enter()
	r.setSource("feature", "main")
	t.Setenv("GIT_EDITOR", "true")
	t.Setenv("GIT_SEQUENCE_EDITOR", "true")
	cmd := dirtyCmd(t, newAcceptCommand(), "carry")
	var branch string
	if err := acceptRun(cmd, []string{"feature"}, acceptSpec(&branch)); err == nil || !state.IsThereActiveRebase() {
		t.Fatalf("err=%v rebase=%v", err, state.IsThereActiveRebase())
	}
	otherTip := r.git("rev-parse", "other")
	featureTip := r.git("rev-parse", "feature")
	cmd = dirtyCmd(t, newAcceptCommand(), "carry")
	buf := captureText(t)
	var asked string
	err := acceptRun(cmd, []string{"other"}, acceptSpec(&asked))
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "will not continue it as 'other'") {
		t.Fatalf("out=%q", buf.String())
	}
	if !state.IsThereActiveRebase() || r.git("rev-parse", "other") != otherTip || r.git("rev-parse", "feature") != featureTip {
		t.Fatalf("rebase %v other %s feature %s", state.IsThereActiveRebase(), r.git("rev-parse", "other"), r.git("rev-parse", "feature"))
	}
	if r.git("rev-parse", "--abbrev-ref", "HEAD") == "main" && r.readFile("f") == "resolved\n" {
		t.Fatal("merged the helper as other")
	}
}

func TestAcceptContinueWithoutNameDeletesTheStartedBranch(t *testing.T) {
	r := newGitRepo(t)
	r.branch("feature", "main")
	r.write("f", "feature\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "feature")
	r.checkout("main")
	r.write("f", "main\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "main")
	r.checkout("feature")
	r.enter()
	r.setSource("feature", "main")
	t.Setenv("GIT_EDITOR", "true")
	t.Setenv("GIT_SEQUENCE_EDITOR", "true")
	cmd := dirtyCmd(t, newAcceptCommand(), "carry")
	var branch string
	if err := acceptRun(cmd, []string{"feature"}, acceptSpec(&branch)); err == nil || !state.IsThereActiveRebase() {
		t.Fatalf("err=%v rebase=%v", err, state.IsThereActiveRebase())
	}
	r.write("f", "resolved\n")
	r.git("add", "f")
	cmd = dirtyCmd(t, newAcceptCommand(), "carry")
	var unnamed string
	if err := acceptRun(cmd, nil, acceptSpec(&unnamed)); err != nil {
		t.Fatal(err)
	}
	if r.git("rev-parse", "--abbrev-ref", "HEAD") != "main" || r.readFile("f") != "resolved\n" {
		t.Fatalf("head %s f %q", r.git("rev-parse", "--abbrev-ref", "HEAD"), r.readFile("f"))
	}
	if r.gitAllow("rev-parse", "--verify", "--quiet", "refs/heads/feature") != "" || r.gitAllow("rev-parse", "--verify", "--quiet", "refs/heads/__eg") != "" {
		t.Fatal("feature or helper remains")
	}
}

func TestAcceptFootprintFromAnotherBranch(t *testing.T) {
	r := newGitRepo(t)
	r.write(".gitignore", "vendor/\n")
	r.git("add", ".gitignore")
	r.git("commit", "-q", "-m", "ignore")
	r.branch("feature", "main")
	if err := os.MkdirAll(filepath.Join(r.dir, "vendor"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.write("vendor/lib.txt", "theirs\n")
	r.git("add", "-f", "vendor/lib.txt")
	r.git("commit", "-q", "-m", "vendor")
	r.checkout("main")
	if err := os.MkdirAll(filepath.Join(r.dir, "vendor"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.write("vendor/lib.txt", "MINE ignored\n")
	r.write("b.txt", "dirty\n")
	r.enter()
	r.setSource("feature", "main")
	head := r.git("rev-parse", "HEAD")
	cmd := dirtyCmd(t, newAcceptCommand(), "carry")
	buf := captureText(t)
	var branch string
	err := acceptRun(cmd, []string{"feature"}, acceptSpec(&branch))
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "vendor/lib.txt would be overwritten") || strings.Contains(buf.String(), "recovery backup was kept") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.git("rev-parse", "HEAD") != head || r.git("rev-parse", "--abbrev-ref", "HEAD") != "main" || r.readFile("vendor/lib.txt") != "MINE ignored\n" || r.readFile("b.txt") != "dirty\n" {
		t.Fatalf("head %s vendor %q b %q", r.git("rev-parse", "--abbrev-ref", "HEAD"), r.readFile("vendor/lib.txt"), r.readFile("b.txt"))
	}
	if r.git("stash", "list") != "" || r.gitAllow("for-each-ref", "refs/elegant-git/recovery") != "" {
		t.Fatal("prepared before refusal")
	}
}

func TestAcceptRefusesStaleHelperBeforeStash(t *testing.T) {
	r := newGitRepo(t)
	r.branch("feature", "main")
	r.commit("feat", "g.txt", "g\n")
	r.git("checkout", "-q", "-B", "__eg", "feature")
	extra := r.commit("extra", "h.txt", "extra\n")
	r.checkout("feature")
	r.write("b.txt", "dirty\n")
	r.enter()
	r.setSource("feature", "main")
	cmd := dirtyCmd(t, newAcceptCommand(), "carry")
	buf := captureText(t)
	var branch string
	err := acceptRun(cmd, []string{"feature"}, acceptSpec(&branch))
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "will not replace") || strings.Contains(buf.String(), "recovery backup was kept") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.git("rev-parse", "__eg") != extra || r.readFile("b.txt") != "dirty\n" || r.git("stash", "list") != "" {
		t.Fatalf("helper %s file %q stash %q", r.git("rev-parse", "__eg"), r.readFile("b.txt"), r.git("stash", "list"))
	}
	if refs := r.gitAllow("for-each-ref", "refs/elegant-git/recovery"); refs != "" {
		t.Fatalf("recovery %s", refs)
	}
}

func TestDiscardLiteralPathspecKeepsStagedAddition(t *testing.T) {
	r := newGitRepo(t)
	r.write("a[1].txt", "tracked\n")
	r.git("add", "a[1].txt")
	r.git("commit", "-q", "-m", "bracket")
	r.branch("other", "main")
	r.checkout("main")
	r.write("a[1].txt", "changed\n")
	r.write("a1.txt", "precious\n")
	r.git("add", "a1.txt")
	r.enter()
	cmd := dirtyCmd(t, newSwitchCommand(), "discard")
	if err := switchRun(cmd, []string{"other"}); err != nil {
		t.Fatal(err)
	}
	if r.readFile("a[1].txt") != "tracked\n" || r.readFile("a1.txt") != "precious\n" {
		t.Fatalf("bracket %q a1 %q", r.readFile("a[1].txt"), r.readFile("a1.txt"))
	}
	if !strings.Contains(r.git("status", "--porcelain"), "?? a1.txt") {
		t.Fatalf("status %s", r.git("status", "--porcelain"))
	}
}

func TestDiscardRefusesStagedAdditionBeforeUnstaging(t *testing.T) {
	r := newGitRepo(t)
	r.branch("other", "main")
	r.checkout("other")
	r.commit("dest", "new.txt", "dest\n")
	r.checkout("main")
	r.write("new.txt", "precious\n")
	r.git("add", "new.txt")
	r.write("f", "changed\n")
	r.enter()
	cmd := dirtyCmd(t, newSwitchCommand(), "discard")
	buf := captureText(t)
	err := switchRun(cmd, []string{"other"})
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "new.txt would be overwritten") || strings.Contains(buf.String(), "already discarded") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.git("rev-parse", "--abbrev-ref", "HEAD") != "main" || r.readFile("new.txt") != "precious\n" {
		t.Fatalf("head %s file %q", r.git("rev-parse", "--abbrev-ref", "HEAD"), r.readFile("new.txt"))
	}
	if !strings.Contains(r.git("diff", "--cached", "--name-only"), "new.txt") {
		t.Fatalf("index %s", r.git("diff", "--cached", "--name-only"))
	}
}

func TestDiscardKeepsEditedStagedAdditionAndRename(t *testing.T) {
	r := newGitRepo(t)
	r.commit("tracked", "tracked", "base\n")
	r.commit("b", "b", "b\n")
	r.branch("other", "main")
	r.checkout("main")
	r.write("tracked", "base\nmod\n")
	r.write("newf", "v1\n")
	r.git("add", "newf")
	r.write("newf", "v2\n")
	r.git("mv", "b", "b2")
	r.write("b2", "b\nz\n")
	r.enter()
	cmd := dirtyCmd(t, newSwitchCommand(), "discard")
	if err := switchRun(cmd, []string{"other"}); err != nil {
		t.Fatal(err)
	}
	if r.git("rev-parse", "--abbrev-ref", "HEAD") != "other" {
		t.Fatalf("head %s", r.git("rev-parse", "--abbrev-ref", "HEAD"))
	}
	if r.readFile("tracked") != "base\n" || r.readFile("b") != "b\n" {
		t.Fatalf("tracked %q b %q", r.readFile("tracked"), r.readFile("b"))
	}
	if r.readFile("newf") != "v2\n" || r.readFile("b2") != "b\nz\n" {
		t.Fatalf("newf %q b2 %q", r.readFile("newf"), r.readFile("b2"))
	}
	status := r.git("status", "--porcelain")
	for _, path := range []string{"newf", "b2"} {
		if !strings.Contains(status, "?? "+path) {
			t.Fatalf("status %q missing %s", status, path)
		}
	}
	if r.gitAllow("ls-files", "--error-unmatch", "newf") != "" || r.gitAllow("ls-files", "--error-unmatch", "b2") != "" {
		t.Fatal("edited additions stayed in the index")
	}
}

func TestDiscardUnstagesDeletedAddition(t *testing.T) {
	r := newGitRepo(t)
	r.branch("other", "main")
	r.checkout("main")
	r.write("added.txt", "x\n")
	r.git("add", "added.txt")
	if err := os.Remove(filepath.Join(r.dir, "added.txt")); err != nil {
		t.Fatal(err)
	}
	r.write("f", "changed\n")
	r.enter()
	cmd := dirtyCmd(t, newSwitchCommand(), "discard")
	if err := switchRun(cmd, []string{"other"}); err != nil {
		t.Fatal(err)
	}
	if r.gitAllow("ls-files", "--error-unmatch", "added.txt") != "" {
		t.Fatal("deleted addition stayed staged")
	}
	if _, err := os.Stat(filepath.Join(r.dir, "added.txt")); !os.IsNotExist(err) {
		t.Fatal("deleted addition came back")
	}
}

func TestSyncDirtySurvivesFetchFailure(t *testing.T) {
	r := newGitRepo(t)
	r.addOrigin("main")
	r.git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing.git"))
	r.branch("feature", "main")
	r.write("f", "dirty\n")
	r.enter()
	r.setSource("feature", "main")
	cmd := dirtyCmd(t, newSyncCommand(), "carry")
	if err := syncRun(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if r.git("rev-parse", "--abbrev-ref", "HEAD") != "feature" || r.readFile("f") != "dirty\n" {
		t.Fatalf("head %s file %q", r.git("rev-parse", "--abbrev-ref", "HEAD"), r.readFile("f"))
	}
}

func TestSaveProtectedNonTTYDoesNotStart(t *testing.T) {
	if os.Getenv("EG_SAVE_DEVNULL") == "1" {
		saveProtectedNonTTYChild()
		return
	}
	r := newGitRepo(t)
	r.write("f", "dirty\n")
	head := r.git("rev-parse", "HEAD")
	cmd := exec.Command(os.Args[0], "-test.run=^TestSaveProtectedNonTTYDoesNotStart$")
	cmd.Dir = r.dir
	cmd.Env = append(os.Environ(), "EG_SAVE_DEVNULL=1", "EG_SAVE_REPO="+r.dir)
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	cmd.Stdin = null
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != exitcode.ProtectedBranch {
		t.Fatalf("exit %v out %s", err, out)
	}
	if !strings.Contains(string(out), "work start") || strings.Contains(string(out), "required input missing") {
		t.Fatalf("out %s", out)
	}
	if r.git("rev-parse", "HEAD") != head || r.readFile("f") != "dirty\n" || r.git("stash", "list") != "" {
		t.Fatalf("head %s file %q stash %q", r.git("rev-parse", "HEAD"), r.readFile("f"), r.git("stash", "list"))
	}
	if got := r.gitAllow("rev-parse", "--verify", "--quiet", "refs/heads/feature"); got != "" {
		t.Fatalf("branch %s", got)
	}
}

func saveProtectedNonTTYChild() {
	cmd := newSaveCommand()
	cmd.SetContext(context.Background())
	var target string
	if err := saveRun(cmd, nil, saveSpec(&target)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func ignoredUpstreamRepo(t *testing.T) (*gitRepo, string) {
	t.Helper()
	r := newGitRepo(t)
	r.write(".gitignore", "vendor/\n")
	r.git("add", ".gitignore")
	r.git("commit", "-q", "-m", "ignore")
	r.addOrigin("main")
	r.branch("feature", "main")
	r.commit("work", "b.txt", "work\n")
	r.checkout("main")
	if err := os.MkdirAll(filepath.Join(r.dir, "vendor"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.write("vendor/lib.txt", "upstream\n")
	r.git("add", "-f", "vendor/lib.txt")
	r.git("commit", "-q", "-m", "vendor")
	r.git("push", "-q", "origin", "main")
	r.checkout("feature")
	if err := os.MkdirAll(filepath.Join(r.dir, "vendor"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.write("vendor/lib.txt", "mine ignored\n")
	r.write("b.txt", "my tracked edit\n")
	return r, r.git("rev-parse", "HEAD")
}

func dirtyCmd(t *testing.T, cmd *cobra.Command, dirty string) *cobra.Command {
	t.Helper()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := cmd.Flags().Set("dirty", dirty); err != nil {
		t.Fatal(err)
	}
	return cmd
}

func refuseIgnored(t *testing.T, r *gitRepo, head, edited string, run func() error) {
	t.Helper()
	buf := captureText(t)
	err := run()
	assertWorkflow(t, err)
	out := buf.String()
	if !strings.Contains(out, "vendor/lib.txt would be overwritten") || strings.Contains(out, "recovery backup was kept") {
		t.Fatalf("out=%q", out)
	}
	if r.git("rev-parse", "HEAD") != head || r.git("rev-parse", "--abbrev-ref", "HEAD") != "feature" {
		t.Fatalf("head %s %s", r.git("rev-parse", "--abbrev-ref", "HEAD"), r.git("rev-parse", "HEAD"))
	}
	if r.readFile(edited) != "my tracked edit\n" || r.readFile("vendor/lib.txt") != "mine ignored\n" || r.git("stash", "list") != "" {
		t.Fatalf("edit %q vendor %q stash %q", r.readFile(edited), r.readFile("vendor/lib.txt"), r.git("stash", "list"))
	}
	if refs := r.gitAllow("for-each-ref", "refs/elegant-git/recovery"); refs != "" {
		t.Fatalf("recovery %s", refs)
	}
}

func (r *gitRepo) readFile(name string) string {
	r.t.Helper()
	body, err := os.ReadFile(filepath.Join(r.dir, name))
	if err != nil {
		r.t.Fatal(err)
	}
	return string(body)
}

func (r *gitRepo) gitAllow(args ...string) string {
	r.t.Helper()
	out, err := r.gitRaw(args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}
