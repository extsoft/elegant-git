package work

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

func requireMergeTree(t *testing.T) {
	t.Helper()
	out, _ := exec.Command(git.Executable(), "merge-tree", "-h").CombinedOutput()
	if !strings.Contains(string(out), "--write-tree") {
		t.Skip("git merge-tree --write-tree requires Git 2.40 or newer")
	}
}

type gitRepo struct {
	t   *testing.T
	dir string
}

func newGitRepo(t *testing.T) *gitRepo {
	t.Helper()
	requireMergeTree(t)
	dir := t.TempDir()
	r := &gitRepo{t: t, dir: dir}
	r.git("init", "-b", "main")
	r.git("config", "cherry-pick.ff", "false")
	r.commit("o", "f", "o\n")
	return r
}

func (r *gitRepo) git(args ...string) string {
	r.t.Helper()
	out, err := r.gitRaw(args...)
	if err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(out)
}

func (r *gitRepo) gitErr(args ...string) error {
	r.t.Helper()
	out, err := r.gitRaw(args...)
	if err != nil {
		return fmt.Errorf("%w\n%s", err, out)
	}
	return nil
}

func (r *gitRepo) gitRaw(args ...string) (string, error) {
	cmd := exec.Command(git.Executable(), args...)
	cmd.Dir = r.dir
	cmd.Env = append(os.Environ(), "GIT_EDITOR=true", "GIT_SEQUENCE_EDITOR=true")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (r *gitRepo) write(name, body string) {
	r.t.Helper()
	if err := os.WriteFile(filepath.Join(r.dir, name), []byte(body), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *gitRepo) commit(msg, name, body string) string {
	r.t.Helper()
	r.write(name, body)
	r.git("add", name)
	r.git("commit", "-q", "-m", msg)
	return r.git("rev-parse", "HEAD")
}

func (r *gitRepo) checkout(branch string) {
	r.t.Helper()
	r.git("checkout", "-q", branch)
}

func (r *gitRepo) branch(name, from string) {
	r.t.Helper()
	r.git("checkout", "-q", "-b", name, from)
}

func (r *gitRepo) squash(branch string) {
	r.t.Helper()
	r.git("merge", "--squash", branch)
	r.git("commit", "-q", "-m", "S")
}

func (r *gitRepo) commitsAB() (a1, a2, b1, b2 string) {
	r.t.Helper()
	r.branch("A", "main")
	a1 = r.commit("a1", "a", "a1\n")
	a2 = r.commit("a2", "b", "a2\n")
	r.branch("B", "A")
	b1 = r.commit("b1", "c", "b1\n")
	b2 = r.commit("b2", "d", "b2\n")
	return
}

func (r *gitRepo) addOrigin(branches ...string) {
	r.t.Helper()
	bare := r.t.TempDir()
	cmd := exec.Command(git.Executable(), "init", "--bare", bare)
	if out, err := cmd.CombinedOutput(); err != nil {
		r.t.Fatalf("bare init: %v\n%s", err, out)
	}
	r.git("remote", "add", "origin", bare)
	r.git(append([]string{"push", "-u", "origin"}, branches...)...)
}

func (r *gitRepo) deleteRemoteHead(branch string) {
	r.t.Helper()
	url := r.git("remote", "get-url", "origin")
	cmd := exec.Command(git.Executable(), "--git-dir", url, "update-ref", "-d", "refs/heads/"+branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		r.t.Fatalf("delete %s on origin: %v\n%s", branch, err, out)
	}
}

func (r *gitRepo) enter() {
	r.t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		r.t.Fatal(err)
	}
	if err := os.Chdir(r.dir); err != nil {
		r.t.Fatal(err)
	}
	git.Use(git.RealRunner{})
	r.t.Cleanup(func() {
		git.Use(git.RealRunner{})
		_ = os.Chdir(wd)
	})
	gitDir := r.git("rev-parse", "--absolute-git-dir")
	r.t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(gitDir, "elegant-git", "state.json"))
}

func (r *gitRepo) setSource(branch, source string) {
	r.t.Helper()
	if err := config.SetBranchSourceBranch(branch, source); err != nil {
		r.t.Fatal(err)
	}
}

func (r *gitRepo) subject(rev string) string {
	r.t.Helper()
	if rev == "" {
		return ""
	}
	return r.git("log", "-1", "--format=%s", rev)
}

func (r *gitRepo) subjects() string {
	r.t.Helper()
	return r.git("log", "--format=%s")
}

func (r *gitRepo) want(branch, target, baseSubject string) {
	r.t.Helper()
	r.checkout(branch)
	gotTarget := config.FreshestBranchSourceBranch(branch)
	if gotTarget != target {
		r.t.Fatalf("target %q, want %q", gotTarget, target)
	}
	gotBase := state.AlreadyAppliedBase(gotTarget)
	if r.subject(gotBase) != baseSubject {
		r.t.Fatalf("base subject %q, want %q", r.subject(gotBase), baseSubject)
	}
}

func TestScenario01ParentUnchanged(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.addOrigin("main", "A", "B")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "origin/A", "")
}

func TestScenario02ParentGainedCommits(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.checkout("A")
	r.commit("a3", "e", "a3\n")
	r.addOrigin("main", "A", "B")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "origin/A", "")
}

func TestScenario03ParentPolished(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.checkout("A")
	r.git("reset", "-q", "--soft", "main")
	r.git("commit", "-q", "-m", "polished")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "A", "a2")
}

func TestScenario04ParentAmended(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.checkout("A")
	r.write("b", "different\n")
	r.git("add", "b")
	r.git("commit", "-q", "--amend", "-m", "a2edited")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "A", "")
}

func TestScenario05ParentLocalOnly(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.enter()
	r.setSource("B", "A")
	r.want("B", "A", "")
}

func TestScenario06ParentRebasedLocally(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.addOrigin("main", "A", "B")
	r.checkout("A")
	r.commit("a3", "e", "a3\n")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "A", "")
}

func TestScenario07FastForwardSameSHAs(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.checkout("main")
	r.git("merge", "--ff-only", "A")
	r.git("branch", "-D", "A")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "main", "")
}

func TestScenario08RebaseMergeNewSHAs(t *testing.T) {
	r := newGitRepo(t)
	a1, a2, _, _ := r.commitsAB()
	r.checkout("main")
	r.commit("unrelated", "u", "u\n")
	r.git("cherry-pick", a1, a2)
	r.git("branch", "-D", "A")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "main", "a2")
}

func TestScenario09MergeCommit(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.checkout("main")
	r.git("merge", "--no-ff", "-m", "merge", "A")
	r.git("branch", "-D", "A")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "main", "")
}

func TestScenario10SquashDeletesBoth(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.addOrigin("main", "A", "B")
	r.checkout("main")
	r.squash("A")
	r.git("push", "origin", "main")
	r.git("push", "origin", "--delete", "A")
	r.checkout("B")
	r.git("branch", "-D", "A")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "origin/main", "a2")
	if err := rebaseOnto("origin/main"); err != nil {
		t.Fatal(err)
	}
	if got := r.subjects(); got != "b2\nb1\nS\no" {
		t.Fatalf("log:\n%s", got)
	}
}

func TestScenario11SquashGoneUpstream(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.addOrigin("main", "A", "B")
	r.checkout("main")
	r.squash("A")
	r.git("push", "origin", "main")
	r.git("push", "origin", "--delete", "A")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "origin/main", "a2")
}

func TestScenario12SquashKeepsRemote(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.addOrigin("main", "A", "B")
	r.checkout("main")
	r.squash("A")
	r.git("push", "origin", "main")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "origin/main", "a2")
}

func TestScenario13SquashNeverPushed(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.checkout("main")
	r.squash("A")
	r.addOrigin("main", "B")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "origin/main", "a2")
}

func TestScenario14SquashIncludesLaterCommit(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.checkout("A")
	r.commit("a3", "e", "a3\n")
	r.checkout("main")
	r.squash("A")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "main", "a2")
}

func TestScenario14SquashOverlapsEarlierLines(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.checkout("A")
	r.commit("a3", "b", "overlap\n")
	r.checkout("main")
	r.squash("A")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "main", "a1")
}

func TestScenario15SquashEdited(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.addOrigin("main", "A", "B")
	r.checkout("main")
	r.git("merge", "--squash", "A")
	r.write("a", "edited\n")
	r.git("add", "a")
	r.git("commit", "-q", "-m", "S")
	r.git("push", "origin", "main")
	r.git("push", "origin", "--delete", "A")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "origin/main", "")
}

func TestScenario16DefaultMovesAfterSquash(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.addOrigin("main", "A", "B")
	r.checkout("main")
	r.squash("A")
	r.commit("later", "a", "later\n")
	r.git("push", "origin", "main")
	r.git("push", "origin", "--delete", "A")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "origin/main", "")
}

func TestScenario17OnlyFirstCommitCherryPicked(t *testing.T) {
	r := newGitRepo(t)
	a1, _, _, _ := r.commitsAB()
	r.checkout("main")
	r.commit("unrelated", "u", "u\n")
	r.git("cherry-pick", a1)
	r.git("branch", "-D", "A")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "main", "a1")
}

func TestScenario18LocalBranchReused(t *testing.T) {
	r := newGitRepo(t)
	r.branch("A", "main")
	r.commit("a1", "a", "a1\n")
	r.commit("a2", "b", "a2\n")
	r.checkout("main")
	r.squash("A")
	r.checkout("A")
	r.commit("a4", "e", "a4\n")
	r.enter()
	r.setSource("A", "main")
	r.want("A", "main", "a2")
}

func TestScenario19NoOwnCommits(t *testing.T) {
	r := newGitRepo(t)
	r.branch("A", "main")
	r.commit("a1", "a", "a1\n")
	r.commit("a2", "b", "a2\n")
	r.branch("B", "A")
	r.checkout("main")
	r.squash("A")
	r.git("branch", "-D", "A")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "main", "a2")
	if err := rebaseOnto("main"); err != nil {
		t.Fatal(err)
	}
	if r.git("rev-parse", "HEAD") != r.git("rev-parse", "main") {
		t.Fatal("branch was not reset to main")
	}
}

func TestScenario20OwnCommitsEditParentLines(t *testing.T) {
	r := newGitRepo(t)
	r.branch("A", "main")
	r.commit("a1", "a", "a1\n")
	r.commit("a2", "b", "a2\n")
	r.branch("B", "A")
	r.write("a", "a1\nb1\n")
	r.git("add", "a")
	r.git("commit", "-q", "-m", "b1")
	r.checkout("main")
	r.squash("A")
	r.git("branch", "-D", "A")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "main", "a2")
	if err := rebaseOnto("main"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(r.dir, "a"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "a1\nb1\n" {
		t.Fatalf("file a = %q", body)
	}
}

func TestScenario21NonPrefixCommitAlreadyUpstream(t *testing.T) {
	r := newGitRepo(t)
	_, _, b1, _ := r.commitsAB()
	r.checkout("main")
	r.git("cherry-pick", b1)
	r.git("branch", "-D", "A")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "main", "")
	if err := rebaseOnto("main"); err != nil {
		t.Fatal(err)
	}
	got := r.git("log", "--format=%s", "main..HEAD")
	if strings.Contains(got, "b1") || !strings.Contains(got, "a1") || !strings.Contains(got, "a2") {
		t.Fatalf("replayed log:\n%s", got)
	}
}

func TestScenario22MergeCommitOnBranch(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.branch("side", "B")
	r.commit("side", "s", "side\n")
	r.checkout("B")
	r.git("merge", "--no-ff", "-m", "merge side", "side")
	r.commit("after", "z", "after\n")
	r.checkout("main")
	r.squash("A")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "main", "a2")
}

func TestScenario23StackParentSquashedChildNotSynced(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.branch("C", "B")
	r.commit("c1", "g", "c1\n")
	r.addOrigin("main", "A", "B", "C")
	r.checkout("main")
	r.squash("A")
	r.git("push", "origin", "main")
	r.enter()
	r.setSource("C", "B")
	r.want("C", "origin/B", "")
}

func TestScenario24StackChildSyncedAndPushed(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.branch("C", "B")
	r.commit("c1", "g", "c1\n")
	r.addOrigin("main", "A", "B", "C")
	r.checkout("main")
	r.squash("A")
	r.git("push", "origin", "main")
	r.checkout("B")
	r.git("rebase", "--onto", "main", "A")
	r.git("push", "--force", "origin", "B")
	r.enter()
	r.setSource("C", "B")
	r.want("C", "origin/B", "b2")
}

func TestScenario25StackChildSyncedLocally(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.branch("C", "B")
	r.commit("c1", "g", "c1\n")
	r.addOrigin("main", "A", "B", "C")
	r.checkout("main")
	r.squash("A")
	r.git("push", "origin", "main")
	r.checkout("B")
	r.git("rebase", "--onto", "main", "A")
	r.enter()
	r.setSource("C", "B")
	r.want("C", "B", "b2")
}

func TestScenario26StackBothSquashed(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.branch("C", "B")
	r.commit("c1", "g", "c1\n")
	r.addOrigin("main", "A", "B", "C")
	r.checkout("main")
	r.squash("B")
	r.git("push", "origin", "main")
	r.git("push", "origin", "--delete", "B")
	r.enter()
	r.setSource("C", "B")
	r.want("C", "origin/main", "b2")
}

func TestScenario27DeliveredSourceWasNotDefault(t *testing.T) {
	r := newGitRepo(t)
	r.branch("X", "main")
	r.commit("x1", "x", "x\n")
	r.branch("A", "X")
	r.commit("a1", "a", "a1\n")
	r.commit("a2", "b", "a2\n")
	r.branch("B", "A")
	r.commit("b1", "c", "b1\n")
	r.checkout("main")
	r.squash("A")
	r.enter()
	r.setSource("A", "X")
	r.setSource("B", "A")
	r.want("B", "main", "a2")
}

func TestScenario28ExplicitSyncRef(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.checkout("main")
	r.squash("A")
	r.enter()
	r.setSource("B", "A")
	r.checkout("B")
	if err := syncLogic("main", false); err != nil {
		t.Fatal(err)
	}
	if got := r.subjects(); got != "b2\nb1\nS\no" {
		t.Fatalf("log:\n%s", got)
	}
}

func TestScenario29RebaseInProgress(t *testing.T) {
	r := newGitRepo(t)
	r.commit("main2", "f", "main\n")
	r.git("checkout", "-q", "-b", "B", "HEAD~1")
	r.commit("b", "f", "b\n")
	if err := r.gitErr("rebase", "main"); err == nil {
		t.Fatal("expected a conflict")
	}
	r.enter()
	if err := syncLogic("", false); err == nil {
		t.Fatal("expected continue to fail")
	}
	if !state.IsThereActiveRebase() {
		t.Fatal("expected the existing rebase to remain")
	}
}

func TestScenario31NoRemotes(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.checkout("main")
	r.squash("A")
	r.enter()
	r.setSource("B", "A")
	r.want("B", "main", "a2")
}

func TestScenario32RemoteBranchLingers(t *testing.T) {
	r := newGitRepo(t)
	r.commitsAB()
	r.addOrigin("main", "A", "B")
	r.checkout("main")
	r.squash("A")
	r.git("push", "origin", "main")
	r.deleteRemoteHead("A")
	if r.git("rev-parse", "--verify", "--quiet", "origin/A") == "" {
		t.Fatal("origin/A should still exist")
	}
	r.enter()
	r.setSource("B", "A")
	r.want("B", "origin/main", "a2")
}

func (r *gitRepo) asidePath(rel string) string {
	r.t.Helper()
	gitDir := r.git("rev-parse", "--absolute-git-dir")
	return filepath.Join(gitDir, "elegant-git", "aside", filepath.FromSlash(rel))
}

func TestRebaseMovesIgnoredVendorAside(t *testing.T) {
	r := newGitRepo(t)
	r.branch("B", "main")
	if err := os.MkdirAll(filepath.Join(r.dir, "third_party", "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.commit("vendor", "third_party/p/CHANGELOG.md", "tracked\n")
	r.git("rm", "-q", "third_party/p/CHANGELOG.md")
	r.write(".gitignore", "third_party/\n")
	r.git("add", ".gitignore")
	r.git("commit", "-q", "-m", "drop")
	r.commit("b1", "c", "b1\n")
	if err := os.MkdirAll(filepath.Join(r.dir, "third_party", "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.write("third_party/p/CHANGELOG.md", "regenerated\n")
	r.enter()
	r.checkout("B")
	if err := rebaseOnto("main"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(r.dir, "third_party", "p", "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "regenerated\n" {
		t.Fatalf("file = %q", body)
	}
	if _, err := os.Stat(r.asidePath("third_party/p/CHANGELOG.md")); !os.IsNotExist(err) {
		t.Fatal("aside file remains")
	}
}

func TestRebaseMovesUntrackedTargetFileAside(t *testing.T) {
	r := newGitRepo(t)
	r.branch("B", "main")
	r.commit("b1", "c", "b1\n")
	r.checkout("main")
	r.commit("kept", "kept", "from-main\n")
	r.checkout("B")
	r.write("kept", "local\n")
	r.enter()
	r.checkout("B")
	if err := rebaseOnto("main"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(r.dir, "kept"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "local\n" {
		t.Fatalf("file = %q", body)
	}
}

func (r *gitRepo) branchWithIgnoredVendorAndConflict() {
	r.t.Helper()
	r.branch("B", "main")
	if err := os.MkdirAll(filepath.Join(r.dir, "third_party", "p"), 0o755); err != nil {
		r.t.Fatal(err)
	}
	r.commit("vendor", "third_party/p/CHANGELOG.md", "tracked\n")
	r.git("rm", "-q", "third_party/p/CHANGELOG.md")
	r.write(".gitignore", "third_party/\n")
	r.git("add", ".gitignore")
	r.git("commit", "-q", "-m", "drop")
	r.write("f", "b\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "conflict")
	r.checkout("main")
	r.write("f", "main\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "main2")
	r.checkout("B")
	if err := os.MkdirAll(filepath.Join(r.dir, "third_party", "p"), 0o755); err != nil {
		r.t.Fatal(err)
	}
	r.write("third_party/p/CHANGELOG.md", "regenerated\n")
}

func TestRebaseRestoresAsideWhenContinueFinishes(t *testing.T) {
	r := newGitRepo(t)
	r.branchWithIgnoredVendorAndConflict()
	r.enter()
	r.checkout("B")
	if err := rebaseOnto("main"); err == nil {
		t.Fatal("expected a conflict")
	}
	if !state.IsThereActiveRebase() {
		t.Fatal("expected the rebase to stay active")
	}
	body, err := os.ReadFile(r.asidePath("third_party/p/CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "regenerated\n" {
		t.Fatalf("aside = %q", body)
	}
	r.write("f", "resolved\n")
	r.git("add", "f")
	t.Setenv("GIT_EDITOR", "true")
	t.Setenv("GIT_SEQUENCE_EDITOR", "true")
	if err := syncLogic("", false); err != nil {
		t.Fatal(err)
	}
	body, err = os.ReadFile(filepath.Join(r.dir, "third_party", "p", "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "regenerated\n" {
		t.Fatalf("file = %q", body)
	}
	aside := filepath.Join(r.git("rev-parse", "--absolute-git-dir"), "elegant-git", "aside")
	if _, err := os.Stat(aside); !os.IsNotExist(err) {
		t.Fatal("aside directory remains")
	}
}

func TestRebaseRestoresLeftoverAsideAfterAbort(t *testing.T) {
	r := newGitRepo(t)
	r.branchWithIgnoredVendorAndConflict()
	r.enter()
	r.checkout("B")
	if err := rebaseOnto("main"); err == nil {
		t.Fatal("expected a conflict")
	}
	r.git("rebase", "--abort")
	if _, err := os.Stat(filepath.Join(r.dir, "third_party", "p", "CHANGELOG.md")); !os.IsNotExist(err) {
		t.Fatal("file should stay aside after abort")
	}
	r.checkout("main")
	if err := rebaseOnto("main"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(r.dir, "third_party", "p", "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "regenerated\n" {
		t.Fatalf("file = %q", body)
	}
}

func TestRebaseRestoresTrackedAsideAfterRebase(t *testing.T) {
	r := newGitRepo(t)
	r.commit("kept", "kept", "from-main\n")
	r.branch("B", "main")
	r.commit("b1", "c", "b1\n")
	r.enter()
	r.checkout("B")
	aside := r.asidePath("kept")
	if err := os.MkdirAll(filepath.Dir(aside), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(aside, []byte("local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := rebaseOnto("main"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(r.dir, "kept"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "local\n" {
		t.Fatalf("file = %q", body)
	}
}

func TestSyncContinueRebasesWithTrackedAside(t *testing.T) {
	r := newGitRepo(t)
	r.branch("B", "main")
	r.write("f", "b\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "conflict")
	r.checkout("main")
	r.commit("kept", "kept", "from-main\n")
	r.write("f", "main\n")
	r.git("add", "f")
	r.git("commit", "-q", "-m", "main2")
	r.checkout("B")
	r.write("kept", "local\n")
	r.enter()
	r.checkout("B")
	if err := rebaseOnto("main"); err == nil {
		t.Fatal("expected a conflict")
	}
	body, err := os.ReadFile(r.asidePath("kept"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "local\n" {
		t.Fatalf("aside = %q", body)
	}
	r.write("f", "resolved\n")
	r.git("add", "f")
	t.Setenv("GIT_EDITOR", "true")
	t.Setenv("GIT_SEQUENCE_EDITOR", "true")
	if err := syncLogic("", false); err != nil {
		t.Fatal(err)
	}
	body, err = os.ReadFile(filepath.Join(r.dir, "kept"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "local\n" {
		t.Fatalf("file = %q", body)
	}
}

func TestPolishMovesIgnoredVendorAside(t *testing.T) {
	r := newGitRepo(t)
	r.branch("B", "main")
	if err := os.MkdirAll(filepath.Join(r.dir, "third_party", "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.commit("vendor", "third_party/p/CHANGELOG.md", "tracked\n")
	r.git("rm", "-q", "third_party/p/CHANGELOG.md")
	r.write(".gitignore", "third_party/\n")
	r.git("add", ".gitignore")
	r.git("commit", "-q", "-m", "drop")
	r.commit("b1", "c", "b1\n")
	if err := os.MkdirAll(filepath.Join(r.dir, "third_party", "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.write("third_party/p/CHANGELOG.md", "regenerated\n")
	r.enter()
	r.checkout("B")
	t.Setenv("GIT_EDITOR", "true")
	t.Setenv("GIT_SEQUENCE_EDITOR", "true")
	if err := polishRun(); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(r.dir, "third_party", "p", "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "regenerated\n" {
		t.Fatalf("file = %q", body)
	}
}

func TestAcceptCheckoutWithUntrackedTargetFile(t *testing.T) {
	r := newGitRepo(t)
	r.branch("B", "main")
	r.commit("b1", "c", "b1\n")
	r.checkout("main")
	r.commit("kept", "kept", "from-main\n")
	r.checkout("B")
	r.write("kept", "local\n")
	r.enter()
	r.checkout("B")
	var branch string
	spec := acceptSpec(&branch)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	if err := acceptRun(cmd, []string{"B"}, spec); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(r.dir, "kept"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "local\n" {
		t.Fatalf("file = %q", body)
	}
}
