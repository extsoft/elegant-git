package repo

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/recovery"
	"github.com/extsoft/elegant-git/internal/text"
)

func TestRealGitPruneLeavesTheOnlySelectedStash(t *testing.T) {
	r := newPruneRepo(t)
	selected := r.stash("selected-ordinary", "selected-body\n")
	buf := capturePrune(t)
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	if !r.hasStash(selected) || !strings.Contains(r.patch(selected), "selected-body") || !strings.Contains(buf.String(), stashDropGuidance) {
		t.Fatalf("stashes\n%s\nout %s", r.stashList(), buf.String())
	}
}

func TestRealGitPruneRefusesWhenAnotherStashIsPresent(t *testing.T) {
	r := newPruneRepo(t)
	keeper := r.stash("keeper", "keeper-body\n")
	selected := r.stash("selected-ordinary", "selected-body\n")
	buf := capturePrune(t)
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), keeper) {
		t.Fatalf("named another stash:\n%s", buf.String())
	}
	if !r.hasStash(selected) || !r.hasStash(keeper) || !strings.Contains(r.patch(keeper), "keeper-body") || !strings.Contains(r.patch(selected), "selected-body") {
		t.Fatalf("stashes\n%s", r.stashList())
	}
}

func TestRealGitPruneABAListDeletesNothing(t *testing.T) {
	r := newPruneRepo(t)
	selected := r.stash("selected-ordinary", "selected-body\n")
	r.git("stash", "drop", "stash@{0}")
	inserted := r.stash("unrelated-external", "external-body\n")
	r.git("stash", "store", "-m", "restored", selected)
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	if !r.hasStash(selected) || !r.hasStash(inserted) {
		t.Fatalf("stashes\n%s", r.stashList())
	}
	if !strings.Contains(r.patch(selected), "selected-body") || !strings.Contains(r.patch(inserted), "external-body") {
		t.Fatalf("patches\n%s", r.stashList())
	}
}

func TestRealGitPruneRefusesWhenSelectionIsTheWholeList(t *testing.T) {
	r := newPruneRepo(t)
	older := r.stash("older", "older-body\n")
	newer := r.stash("newer", "newer-body\n")
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	if !r.hasStash(older) || !r.hasStash(newer) || !strings.Contains(r.patch(older), "older-body") || !strings.Contains(r.patch(newer), "newer-body") {
		t.Fatalf("stashes\n%s", r.stashList())
	}
}

func TestRealGitPruneRefusesAPartialSelection(t *testing.T) {
	r := newPruneRepo(t)
	oldest := r.stash("oldest", "oldest-body\n")
	keeper := r.stash("keeper", "keeper-body\n")
	newest := r.stash("newest", "newest-body\n")
	buf := capturePrune(t)
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), keeper) {
		t.Fatalf("named another stash:\n%s", buf.String())
	}
	if !r.hasStash(newest) || !r.hasStash(keeper) || !r.hasStash(oldest) {
		t.Fatalf("stashes\n%s", r.stashList())
	}
	if !strings.Contains(r.patch(keeper), "keeper-body") || !strings.Contains(r.patch(newest), "newest-body") || !strings.Contains(r.patch(oldest), "oldest-body") {
		t.Fatalf("patches\n%s", r.stashList())
	}
}

func TestRealGitPruneRefusesDuplicateStashIdentity(t *testing.T) {
	r := newPruneRepo(t)
	oid := r.stash("original", "original-body\n")
	r.stash("other", "other-body\n")
	r.git("stash", "store", "-m", "duplicate", oid)
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	if !r.hasStash(oid) || !strings.Contains(r.patch(oid), "original-body") {
		t.Fatalf("stashes\n%s", r.stashList())
	}
}

func TestRealGitPruneKeepsStashWhoseSubjectContainsATab(t *testing.T) {
	r := newPruneRepo(t)
	tabbed := r.stash("weird\tname \"quoted\"", "tab-body\n")
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	if !r.hasStash(tabbed) || !strings.Contains(r.patch(tabbed), "tab-body") {
		t.Fatalf("stashes\n%s", r.stashList())
	}
}

func TestRealGitPruneKeepsRecoveryStashWhenItsNumberChanges(t *testing.T) {
	r := newPruneRepo(t)
	owned := r.stash("owned", "owned-body\n")
	journal := r.saveRecovery(t, owned)
	selected := r.stash("selected-ordinary", "selected-body\n")
	buf := capturePrune(t)
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), owned) || !strings.Contains(buf.String(), "selected-ordinary") {
		t.Fatalf("out=%s", buf.String())
	}
	if !r.hasStash(owned) || !r.hasStash(selected) || !strings.Contains(r.patch(owned), "owned-body") || !strings.Contains(r.patch(selected), "selected-body") {
		t.Fatalf("stashes\n%s", r.stashList())
	}
	if got := r.journal(t); got != journal {
		t.Fatalf("journal changed\n%s", got)
	}
}

type pruneGit struct {
	t   *testing.T
	dir string
}

func newPruneRepo(t *testing.T) *pruneGit {
	t.Helper()
	unsetPruneGitEnv(t)
	dir := t.TempDir()
	r := &pruneGit{t: t, dir: dir}
	r.git("init", "-q", "-b", "main")
	r.git("config", "user.email", "t@example.com")
	r.git("config", "user.name", "Test")
	r.write("f", "base\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "base")
	t.Chdir(dir)
	git.Use(git.RealRunner{})
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	return r
}

func (r *pruneGit) stash(msg, body string) string {
	r.t.Helper()
	r.write("f", body)
	r.git("stash", "push", "-q", "-m", msg)
	return r.git("rev-parse", "stash@{0}")
}

func (r *pruneGit) saveRecovery(t *testing.T, oid string) string {
	t.Helper()
	if err := recovery.Save(r.common(t), recovery.File{Ops: []recovery.Op{{
		ID: "abc", Command: "work.sync", Phase: recovery.PhaseSuspended, StashOID: oid,
		Ref: recovery.RefName("abc"),
	}}}); err != nil {
		t.Fatal(err)
	}
	return r.journal(t)
}

func (r *pruneGit) common(t *testing.T) string {
	t.Helper()
	common := r.git("rev-parse", "--git-common-dir")
	if !filepath.IsAbs(common) {
		common = filepath.Join(r.dir, common)
	}
	return common
}

func (r *pruneGit) journal(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(r.common(t), "elegant-git", "recovery.json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func (r *pruneGit) hasStash(oid string) bool {
	r.t.Helper()
	for _, line := range strings.Split(r.stashList(), "\n") {
		if strings.TrimSpace(line) == oid {
			return true
		}
	}
	return false
}

func (r *pruneGit) stashList() string {
	r.t.Helper()
	return r.git("stash", "list", "--format=%H")
}

func (r *pruneGit) patch(oid string) string {
	r.t.Helper()
	return r.git("stash", "show", "-p", oid)
}

func (r *pruneGit) git(args ...string) string {
	r.t.Helper()
	cmd := exec.Command(git.Executable(), args...)
	cmd.Dir = r.dir
	cmd.Env = append(pruneGitEnv(os.Environ()), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=t@example.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func (r *pruneGit) write(name, body string) {
	r.t.Helper()
	if err := os.WriteFile(filepath.Join(r.dir, name), []byte(body), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func unsetPruneGitEnv(t *testing.T) {
	t.Helper()
	prev := map[string]string{}
	for _, key := range pruneGitEnvKeys {
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

func pruneGitEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		skip := false
		for _, name := range pruneGitEnvKeys {
			if key == name {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, entry)
		}
	}
	return out
}

var pruneGitEnvKeys = []string{
	"GIT_DIR",
	"GIT_WORK_TREE",
	"GIT_INDEX_FILE",
	"GIT_COMMON_DIR",
	"GIT_PREFIX",
	"GIT_OBJECT_DIRECTORY",
	"GIT_ALTERNATE_OBJECT_DIRECTORIES",
	"GIT_NAMESPACE",
}

func (r *pruneGit) head() string {
	return r.git("rev-parse", "--abbrev-ref", "HEAD")
}

func (r *pruneGit) gitAllow(args ...string) (string, error) {
	r.t.Helper()
	cmd := exec.Command(git.Executable(), args...)
	cmd.Dir = r.dir
	cmd.Env = append(pruneGitEnv(os.Environ()), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=t@example.com")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runPruneGit(t *testing.T, p prompt.Prompter, dirty string, args ...string) error {
	t.Helper()
	cmd := newPruneCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), p))
	if dirty != "" {
		if err := cmd.Flags().Set("dirty", dirty); err != nil {
			t.Fatal(err)
		}
	}
	return pruneRun(cmd, args)
}

func capturePrune(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	return buf
}

func TestPruneDirtyNoninteractiveRefusesBeforeCheckout(t *testing.T) {
	r := newPruneRepo(t)
	r.git("checkout", "-q", "-b", "feature")
	r.write("f", "dirty\n")
	buf := capturePrune(t)
	err := runPruneGit(t, prompt.NewNonInteractive(), "", "local")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "--dirty") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.head() != "feature" || r.read("f") != "dirty\n" || r.stashList() != "" {
		t.Fatalf("head %s stash %q", r.head(), r.stashList())
	}
}

func TestPruneCancelBeforePreparationChangesNothing(t *testing.T) {
	r := newPruneRepo(t)
	r.git("checkout", "-q", "-b", "feature")
	r.write("f", "dirty\n")
	err := runPruneGit(t, prompt.NewNonInteractive(), "cancel", "local")
	if err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("err=%v", err)
	}
	if r.head() != "feature" || r.read("f") != "dirty\n" || r.stashList() != "" {
		t.Fatalf("head %s stash %q", r.head(), r.stashList())
	}
}

func TestPrunePhaseListCancelLeavesADirtyTree(t *testing.T) {
	r := newPruneRepo(t)
	r.git("checkout", "-q", "-b", "feature")
	r.write("f", "dirty\n")
	p := newPrunePrompter(prunePickResult{err: prompt.ErrUserCancelled})
	err := runPruneGit(t, p, "")
	if !errors.Is(err, prompt.ErrUserCancelled) {
		t.Fatalf("err=%v", err)
	}
	if r.head() != "feature" || r.read("f") != "dirty\n" || r.stashList() != "" {
		t.Fatalf("head %s stash %q", r.head(), r.stashList())
	}
}

func TestPruneCarryLandsOnTheDefaultBranch(t *testing.T) {
	r := newPruneRepo(t)
	r.git("checkout", "-q", "-b", "feature")
	r.write("g", "kept\n")
	r.git("add", "g")
	r.git("commit", "-q", "-m", "keep feature")
	r.write("f", "staged\n")
	r.git("add", "f")
	r.write("f", "unstaged\n")
	r.write("extra", "u\n")
	cached := r.git("diff", "--cached")
	plain := r.git("diff")
	if err := runPruneGit(t, prompt.NewNonInteractive(), "carry", "worktrees"); err != nil {
		t.Fatal(err)
	}
	if r.head() != "main" || r.git("diff", "--cached") != cached || r.git("diff") != plain || r.read("extra") != "u\n" {
		t.Fatalf("head %s cached %q plain %q", r.head(), r.git("diff", "--cached"), r.git("diff"))
	}
	if _, err := os.Stat(filepath.Join(r.dir, ".git", "refs", "heads", "feature")); err != nil {
		t.Fatal("source branch was deleted")
	}
}

func TestPruneCarryKeepsBackupWhenSourceIsDeleted(t *testing.T) {
	r := newPruneRepo(t)
	r.git("checkout", "-q", "-b", "feature")
	r.write("f", "dirty\n")
	buf := capturePrune(t)
	if err := runPruneGit(t, prompt.NewNonInteractive(), "carry", "local"); err != nil {
		t.Fatal(err)
	}
	if r.head() != "main" || r.read("f") != "base\n" {
		t.Fatalf("head %s file %q", r.head(), r.read("f"))
	}
	out := buf.String()
	if !strings.Contains(out, "feature") || !strings.Contains(out, "was kept") || strings.Contains(out, "will not be applied again") {
		t.Fatalf("out=%q", out)
	}
	if _, err := r.gitAllow("rev-parse", "--verify", "--quiet", "refs/heads/feature"); err == nil {
		t.Fatal("source branch still exists")
	}
	if r.stashList() == "" || !strings.Contains(r.journal(t), "feature") {
		t.Fatalf("stash %q journal %s", r.stashList(), r.journal(t))
	}
	err := runPruneGit(t, prompt.NewNonInteractive(), "", "local")
	if err == nil || !strings.Contains(err.Error(), "repo doctor") {
		t.Fatalf("second err=%v", err)
	}
	if r.head() != "main" || r.read("f") != "base\n" {
		t.Fatalf("second head %s file %q", r.head(), r.read("f"))
	}
	if _, err := r.gitAllow("rev-parse", "--verify", "--quiet", "refs/heads/feature"); err == nil {
		t.Fatal("second run restored the source branch")
	}
}

func TestPrunePhaseCancelStillCarries(t *testing.T) {
	r := newPruneRepo(t)
	r.git("checkout", "-q", "-b", "old")
	r.git("checkout", "-q", "main")
	r.git("checkout", "-q", "-b", "feature")
	r.write("g", "kept\n")
	r.git("add", "g")
	r.git("commit", "-q", "-m", "keep feature")
	r.write("f", "dirty\n")
	buf := capturePrune(t)
	p := newPrunePrompter(prunePickResult{err: prompt.ErrUserCancelled})
	if err := runPruneGit(t, p, "carry", "local"); err != nil {
		t.Fatal(err)
	}
	if r.head() != "main" || r.read("f") != "dirty\n" {
		t.Fatalf("head %s file %q", r.head(), r.read("f"))
	}
	if _, err := os.Stat(filepath.Join(r.dir, ".git", "refs", "heads", "old")); err != nil {
		t.Fatal("cancelled phase deleted a branch")
	}
	out := buf.String()
	if strings.Contains(out, "no longer exists") || strings.Contains(out, "repo prune cancelled") {
		t.Fatalf("out=%q", out)
	}
}

func TestPruneDiscardRefusesStagedAdditionFromUpstream(t *testing.T) {
	r := newPruneRepo(t)
	remote := t.TempDir()
	r.git("init", "--bare", "-q", remote)
	r.git("remote", "add", "origin", remote)
	r.git("push", "-q", "-u", "origin", "HEAD:main")
	other := t.TempDir()
	gitIn(t, other, "clone", "-q", remote, ".")
	gitIn(t, other, "config", "user.email", "t@example.com")
	gitIn(t, other, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(other, "P"), []byte("upstream\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, other, "add", "P")
	gitIn(t, other, "commit", "-q", "-m", "add P")
	gitIn(t, other, "push", "-q", "origin", "HEAD:main")
	r.git("checkout", "-q", "-b", "feature")
	r.write("f", "dirty\n")
	r.write("P", "mine\n")
	r.git("add", "P")
	buf := capturePrune(t)
	err := runPruneGit(t, prompt.NewNonInteractive(), "discard", "worktrees")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "P") || !strings.Contains(buf.String(), "left in place") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.head() != "feature" || r.read("f") != "dirty\n" || r.read("P") != "mine\n" {
		t.Fatalf("head %s f %q P %q", r.head(), r.read("f"), r.read("P"))
	}
	if !strings.Contains(r.git("diff", "--cached", "--name-only"), "P") || r.stashList() != "" {
		t.Fatalf("cached %s stash %q", r.git("diff", "--cached", "--name-only"), r.stashList())
	}
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command(git.Executable(), args...)
	cmd.Dir = dir
	cmd.Env = append(pruneGitEnv(os.Environ()), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=t@example.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestPruneDiscardKeepsUntrackedAndRefusesObstructions(t *testing.T) {
	t.Run("tracked edit and untracked file", func(t *testing.T) {
		r := newPruneRepo(t)
		r.git("checkout", "-q", "-b", "feature")
		r.write("g", "kept\n")
		r.git("add", "g")
		r.git("commit", "-q", "-m", "keep feature")
		r.write("f", "dirty\n")
		r.write("extra", "stay\n")
		r.write("added", "new\n")
		r.git("add", "added")
		if err := runPruneGit(t, prompt.NewNonInteractive(), "discard", "worktrees"); err != nil {
			t.Fatal(err)
		}
		if r.head() != "main" || r.read("f") != "base\n" || r.read("extra") != "stay\n" || r.read("added") != "new\n" {
			t.Fatalf("head %s f %q extra %q added %q", r.head(), r.read("f"), r.read("extra"), r.read("added"))
		}
		if !strings.Contains(r.git("status", "--porcelain"), "?? added") || !strings.Contains(r.git("status", "--porcelain"), "?? extra") {
			t.Fatalf("status %s", r.git("status", "--porcelain"))
		}
	})
	t.Run("untracked file the default branch tracks", func(t *testing.T) {
		r := newPruneRepo(t)
		r.write("g", "tracked\n")
		r.git("add", "g")
		r.git("commit", "-q", "-m", "track g")
		r.git("checkout", "-q", "-b", "feature")
		r.git("rm", "g")
		r.git("commit", "-q", "-m", "drop g")
		r.write("g", "mine\n")
		r.write("f", "dirty\n")
		buf := capturePrune(t)
		err := runPruneGit(t, prompt.NewNonInteractive(), "discard", "local")
		assertWorkflow(t, err)
		if !strings.Contains(buf.String(), "g") || !strings.Contains(buf.String(), "left in place") {
			t.Fatalf("out=%q", buf.String())
		}
		if r.head() != "feature" || r.read("g") != "mine\n" || r.read("f") != "dirty\n" || r.stashList() != "" {
			t.Fatalf("head %s g %q", r.head(), r.read("g"))
		}
	})
}

func TestPruneIgnoredAndUntrackedCollisionsAreRefused(t *testing.T) {
	t.Run("ignored", func(t *testing.T) {
		r := newPruneRepo(t)
		r.write("x.log", "tracked\n")
		r.git("add", "x.log")
		r.git("commit", "-q", "-m", "log")
		r.git("checkout", "-q", "-b", "feature")
		r.git("rm", "x.log")
		r.write(".gitignore", "*.log\n")
		r.git("add", ".gitignore")
		r.git("commit", "-q", "-m", "ignore")
		r.write("x.log", "mine\n")
		buf := capturePrune(t)
		err := runPruneGit(t, prompt.NewNonInteractive(), "", "local")
		assertWorkflow(t, err)
		if !strings.Contains(buf.String(), "x.log") {
			t.Fatalf("out=%q", buf.String())
		}
		if r.head() != "feature" || r.read("x.log") != "mine\n" || r.stashList() != "" {
			t.Fatalf("head %s stash %q", r.head(), r.stashList())
		}
	})
	t.Run("carry untracked", func(t *testing.T) {
		r := newPruneRepo(t)
		r.write("g", "tracked\n")
		r.git("add", "g")
		r.git("commit", "-q", "-m", "track g")
		r.git("checkout", "-q", "-b", "feature")
		r.git("rm", "g")
		r.git("commit", "-q", "-m", "drop g")
		r.write("g", "mine\n")
		buf := capturePrune(t)
		err := runPruneGit(t, prompt.NewNonInteractive(), "carry", "local")
		assertWorkflow(t, err)
		if !strings.Contains(buf.String(), "g") {
			t.Fatalf("out=%q", buf.String())
		}
		if r.head() != "feature" || r.read("g") != "mine\n" || r.stashList() != "" {
			t.Fatalf("head %s stash %q", r.head(), r.stashList())
		}
	})
}

func TestPruneStashLeavesAnOrdinaryStash(t *testing.T) {
	r := newPruneRepo(t)
	r.git("checkout", "-q", "-b", "feature")
	r.write("g", "kept\n")
	r.git("add", "g")
	r.git("commit", "-q", "-m", "keep feature")
	r.write("f", "dirty\n")
	buf := capturePrune(t)
	if err := runPruneGit(t, prompt.NewNonInteractive(), "stash", "worktrees"); err != nil {
		t.Fatal(err)
	}
	if r.head() != "main" || r.read("f") != "base\n" || r.stashList() == "" {
		t.Fatalf("head %s file %q stash %q", r.head(), r.read("f"), r.stashList())
	}
	if !strings.Contains(buf.String(), "was not applied") || !strings.Contains(r.git("stash", "list"), "WIP on feature") {
		t.Fatalf("out=%q list=%s", buf.String(), r.git("stash", "list"))
	}
}

func TestPruneHidesRecoveryStashAfterPreparationShiftsIt(t *testing.T) {
	r := newPruneRepo(t)
	owned := r.stash("owned", "owned-body\n")
	r.git("update-ref", "refs/elegant-git/recovery/abc", owned)
	ordinary := r.stash("ordinary", "ordinary-body\n")
	r.git("checkout", "-q", "-b", "feature")
	r.write("g", "kept\n")
	r.git("add", "g")
	r.git("commit", "-q", "-m", "keep feature")
	r.write("f", "dirty\n")
	buf := capturePrune(t)
	if err := runPruneGit(t, prompt.NewNonInteractive(), "carry", "stashes"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "owned") || strings.Contains(out, owned) || !strings.Contains(out, "ordinary") || !strings.Contains(out, stashDropGuidance) {
		t.Fatalf("out=%s", out)
	}
	if r.head() != "main" || r.read("f") != "dirty\n" || !r.hasStash(owned) || !r.hasStash(ordinary) {
		t.Fatalf("head %s f %q stashes %s", r.head(), r.read("f"), r.stashList())
	}
	if strings.Contains(out, "was kept") || strings.Contains(out, "nothing was dropped") || strings.Contains(out, "while "+owned) {
		t.Fatalf("out=%s", out)
	}
}

func TestPruneRefusesRebaseBeforeDirtyPreparation(t *testing.T) {
	r := newPruneRepo(t)
	r.write("f", "main\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "main")
	r.git("checkout", "-q", "-b", "feature")
	r.write("f", "feature\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "feature")
	r.git("checkout", "-q", "main")
	r.write("f", "other\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "other")
	r.git("checkout", "-q", "feature")
	if _, err := r.gitAllow("rebase", "main"); err == nil {
		t.Fatal("expected a conflict")
	}
	r.write("f", "resolved\n")
	r.git("add", "f")
	buf := capturePrune(t)
	err := runPruneGit(t, prompt.NewNonInteractive(), "carry", "local")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "rebase") || r.stashList() != "" {
		t.Fatalf("out=%q stash %q", buf.String(), r.stashList())
	}
	if _, statErr := os.Stat(filepath.Join(r.dir, ".git", "rebase-merge")); statErr != nil {
		t.Fatal("rebase was cleared")
	}
	if !strings.Contains(r.git("diff", "--cached"), "resolved") {
		t.Fatalf("cached %s", r.git("diff", "--cached"))
	}
}

func TestPruneInvalidDirtyBeforePhaseList(t *testing.T) {
	r := newPruneRepo(t)
	r.git("checkout", "-q", "-b", "feature")
	p := newPrunePrompter(prunePickResult{})
	err := runPruneGit(t, p, "nope")
	var usage *cliruntime.UsageError
	if !errors.As(err, &usage) || !strings.Contains(err.Error(), "not carry") || len(p.calls) != 0 {
		t.Fatalf("err=%v calls=%d", err, len(p.calls))
	}
	if r.head() != "feature" {
		t.Fatalf("head %s", r.head())
	}
}

func TestPruneStashNameBeforeEmptyPhaseList(t *testing.T) {
	r := newPruneRepo(t)
	p := newPrunePrompter(prunePickResult{})
	cmd := newPruneCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), p))
	if err := cmd.Flags().Set("stash-name", "wip"); err != nil {
		t.Fatal(err)
	}
	err := pruneRun(cmd, nil)
	var usage *cliruntime.UsageError
	if !errors.As(err, &usage) || !strings.Contains(err.Error(), "--stash-name") || len(p.calls) != 0 {
		t.Fatalf("err=%v calls=%d", err, len(p.calls))
	}
	if r.head() != "main" {
		t.Fatalf("head %s", r.head())
	}
}

func TestPruneSuspendedBackupRefusesBeforeFetch(t *testing.T) {
	r := newPruneRepo(t)
	r.write(".gitignore", "*.log\n")
	r.git("add", ".gitignore")
	r.git("commit", "-q", "-m", "ignore")
	remote := t.TempDir()
	r.git("init", "--bare", "-q", remote)
	r.git("remote", "add", "origin", remote)
	r.git("push", "-q", "-u", "origin", "HEAD:main")
	r.git("checkout", "-q", "-b", "feature")
	r.write("f", "dirty\n")
	if err := runPruneGit(t, prompt.NewNonInteractive(), "carry", "local"); err != nil {
		t.Fatal(err)
	}
	r.write("x.log", "ignored\n")
	r.git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing"))
	buf := capturePrune(t)
	err := runPruneGit(t, prompt.NewNonInteractive(), "", "local")
	if err == nil || !strings.Contains(err.Error(), "repo doctor") {
		t.Fatalf("err=%v", err)
	}
	if strings.Contains(buf.String(), "git fetch") || strings.Contains(buf.String(), "can't be fetched") {
		t.Fatalf("fetched before refusal: %s", buf.String())
	}
	if r.head() != "main" || r.read("f") != "base\n" || r.read("x.log") != "ignored\n" {
		t.Fatalf("head %s f %q log %q", r.head(), r.read("f"), r.read("x.log"))
	}
}

type stashNameCancel struct{ prompt.Prompter }

func (stashNameCancel) Pick(string, []prompt.Choice, string) (string, error) {
	return "stash", nil
}

func (stashNameCancel) String(string, string) (string, error) {
	return "", prompt.ErrUserCancelled
}

func TestPruneStashNameCancelDoesNotFetch(t *testing.T) {
	r := newPruneRepo(t)
	remote := t.TempDir()
	r.git("init", "--bare", "-q", remote)
	r.git("remote", "add", "origin", remote)
	r.git("push", "-q", "-u", "origin", "HEAD:main")
	r.git("checkout", "-q", "-b", "gone")
	r.git("push", "-q", "origin", "gone")
	r.git("checkout", "-q", "main")
	r.git("branch", "-D", "gone")
	r.git("checkout", "-q", "-b", "feature")
	r.write("f", "dirty\n")
	before := r.git("rev-parse", "refs/remotes/origin/gone")
	buf := capturePrune(t)
	err := runPruneGit(t, stashNameCancel{Prompter: prompt.NewNonInteractive()}, "", "local")
	if err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("err=%v", err)
	}
	if got := r.git("rev-parse", "refs/remotes/origin/gone"); got != before || strings.Contains(buf.String(), "git fetch") {
		t.Fatalf("gone %s before %s\nout %s", got, before, buf.String())
	}
	if r.head() != "feature" || r.read("f") != "dirty\n" || r.stashList() != "" {
		t.Fatalf("head %s f %q stash %q", r.head(), r.read("f"), r.stashList())
	}
}

func TestPruneCarryRefusesStagedAdditionFromUpstream(t *testing.T) {
	r := newPruneRepo(t)
	remote := t.TempDir()
	r.git("init", "--bare", "-q", remote)
	r.git("remote", "add", "origin", remote)
	r.git("push", "-q", "-u", "origin", "HEAD:main")
	other := t.TempDir()
	gitIn(t, other, "clone", "-q", remote, ".")
	gitIn(t, other, "config", "user.email", "t@example.com")
	gitIn(t, other, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(other, "P"), []byte("upstream\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, other, "add", "P")
	gitIn(t, other, "commit", "-q", "-m", "add P")
	gitIn(t, other, "push", "-q", "origin", "HEAD:main")
	r.git("checkout", "-q", "-b", "feature")
	r.write("f", "dirty\n")
	r.write("P", "mine\n")
	r.git("add", "P")
	buf := capturePrune(t)
	err := runPruneGit(t, prompt.NewNonInteractive(), "carry", "worktrees")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "P") || !strings.Contains(buf.String(), "left in place") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.head() != "feature" || r.read("f") != "dirty\n" || r.read("P") != "mine\n" {
		t.Fatalf("head %s f %q P %q", r.head(), r.read("f"), r.read("P"))
	}
	if !strings.Contains(r.git("diff", "--cached", "--name-only"), "P") || r.stashList() != "" {
		t.Fatalf("index %q stash %q", r.git("diff", "--cached", "--name-only"), r.stashList())
	}
}

func TestPruneCarryRestoresAfterPhaseError(t *testing.T) {
	r := newPruneRepo(t)
	r.git("checkout", "-q", "-b", "held")
	r.git("checkout", "-q", "main")
	wt := t.TempDir()
	r.git("worktree", "add", "--quiet", wt, "held")
	r.git("checkout", "-q", "-b", "feature")
	r.write("g", "kept\n")
	r.git("add", "g")
	r.git("commit", "-q", "-m", "keep feature")
	r.write("f", "dirty\n")
	buf := capturePrune(t)
	err := runPruneGit(t, prompt.NewNonInteractive(), "carry", "local")
	if err == nil || strings.Contains(err.Error(), "repo doctor") || strings.Contains(buf.String(), "was kept") {
		t.Fatalf("err=%v out=%s", err, buf.String())
	}
	if r.head() != "main" || r.read("f") != "dirty\n" {
		t.Fatalf("head %s f %q", r.head(), r.read("f"))
	}
	if strings.Contains(buf.String(), "nothing was dropped") || strings.Contains(buf.String(), "while ") {
		t.Fatalf("out=%s", buf.String())
	}
	if data, readErr := os.ReadFile(filepath.Join(r.common(t), "elegant-git", "recovery.json")); readErr == nil && strings.Contains(string(data), "suspended") {
		t.Fatalf("journal %s", data)
	}
}

func TestPruneStashPhaseErrorKeepsTheBackup(t *testing.T) {
	r := newPruneRepo(t)
	r.git("checkout", "-q", "-b", "held")
	r.git("checkout", "-q", "main")
	wt := t.TempDir()
	r.git("worktree", "add", "--quiet", wt, "held")
	r.git("checkout", "-q", "-b", "feature")
	r.write("g", "kept\n")
	r.git("add", "g")
	r.git("commit", "-q", "-m", "keep feature")
	r.write("f", "dirty\n")
	buf := capturePrune(t)
	err := runPruneGit(t, prompt.NewNonInteractive(), "stash", "local")
	if err == nil || !strings.Contains(err.Error()+"\n"+buf.String(), "was kept") {
		t.Fatalf("err=%v out=%s", err, buf.String())
	}
	if r.head() != "main" || r.read("f") != "base\n" || r.stashList() == "" {
		t.Fatalf("head %s f %q stash %q", r.head(), r.read("f"), r.stashList())
	}
	if !strings.Contains(r.journal(t), "suspended") {
		t.Fatalf("journal %s", r.journal(t))
	}
	again := runPruneGit(t, prompt.NewNonInteractive(), "", "local")
	if again == nil || !strings.Contains(again.Error(), "repo doctor") {
		t.Fatalf("again=%v", again)
	}
}

func (r *pruneGit) read(name string) string {
	r.t.Helper()
	data, err := os.ReadFile(filepath.Join(r.dir, name))
	if err != nil {
		r.t.Fatal(err)
	}
	return string(data)
}
