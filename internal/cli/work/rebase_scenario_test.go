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
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/state"
)

func mergeTreeHelpOK(help string) bool {
	return strings.Contains(help, "--write-tree") && strings.Contains(help, "merge-base")
}

func requireMergeTree(t *testing.T) {
	t.Helper()
	out, _ := exec.Command(git.Executable(), "merge-tree", "-h").CombinedOutput()
	if !mergeTreeHelpOK(string(out)) {
		t.Fatalf("required Git 2.40 interface merge-tree --write-tree --merge-base is missing from %s", git.Executable())
	}
}

func TestMergeTreeHelpRejectsWriteTreeAlone(t *testing.T) {
	if mergeTreeHelpOK("usage: --write-tree\n") {
		t.Fatal("--write-tree alone is not the required interface")
	}
	if !mergeTreeHelpOK("--write-tree\n--merge-base <commit>\n") {
		t.Fatal("expected both options")
	}
	if !mergeTreeHelpOK("--write-tree\n--[no-]merge-base <tree-ish>\n") {
		t.Fatal("expected the bracketed help form")
	}
}

func TestMergeTreeExplicitBase(t *testing.T) {
	r := newGitRepo(t)
	base := r.git("rev-parse", "HEAD")
	r.git("checkout", "-q", "-b", "left")
	r.write("left", "L\n")
	r.git("add", "left")
	r.git("commit", "-q", "-m", "L")
	left := r.git("rev-parse", "HEAD")
	r.git("checkout", "-q", "main")
	r.git("checkout", "-q", "-b", "right")
	r.write("right", "R\n")
	r.git("add", "right")
	r.git("commit", "-q", "-m", "R")
	right := r.git("rev-parse", "HEAD")
	out := r.git("merge-tree", "--write-tree", "--merge-base="+base, left, right)
	tree := strings.TrimSpace(strings.Split(out, "\n")[0])
	if r.git("cat-file", "-t", tree) != "tree" {
		t.Fatalf("merge-tree output %q", out)
	}
	files := r.git("ls-tree", "-r", "--name-only", tree)
	if !strings.Contains(files, "left") || !strings.Contains(files, "right") {
		t.Fatalf("tree files %q", files)
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
	buf := captureText(t)
	err := rebaseOnto("main")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "left in place") {
		t.Fatalf("out=%q", buf.String())
	}
	body, err := os.ReadFile(filepath.Join(r.dir, "third_party", "p", "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "regenerated\n" {
		t.Fatalf("file = %q", body)
	}
	if _, err := os.Stat(r.asidePath("third_party/p/CHANGELOG.md")); !os.IsNotExist(err) {
		t.Fatal("ignored file was moved aside")
	}
}

func TestRebaseRefusesIgnoredFromSubdirectory(t *testing.T) {
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
	sub := filepath.Join(r.dir, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}
	buf := captureText(t)
	err := rebaseOnto("main")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "left in place") {
		t.Fatalf("out=%q", buf.String())
	}
	body, err := os.ReadFile(filepath.Join(r.dir, "third_party", "p", "CHANGELOG.md"))
	if err != nil || string(body) != "regenerated\n" {
		t.Fatalf("file %q err=%v", body, err)
	}
	if _, statErr := os.Stat(r.asidePath("third_party/p/CHANGELOG.md")); !os.IsNotExist(statErr) {
		t.Fatal("ignored file was moved aside")
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
	err := rebaseOnto("main")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err=%v", err)
	}
	body, err := os.ReadFile(filepath.Join(r.dir, "kept"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "from-main\n" {
		t.Fatalf("file = %q", body)
	}
	aside, err := os.ReadFile(r.asidePath("kept"))
	if err != nil || string(aside) != "local\n" {
		t.Fatalf("aside = %q err=%v", aside, err)
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
	buf := captureText(t)
	err := rebaseOnto("main")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "left in place") {
		t.Fatalf("out=%q", buf.String())
	}
	if state.IsThereActiveRebase() {
		t.Fatal("rebase started")
	}
	body, err := os.ReadFile(filepath.Join(r.dir, "third_party", "p", "CHANGELOG.md"))
	if err != nil || string(body) != "regenerated\n" {
		t.Fatalf("file %q err=%v", body, err)
	}
	if _, err := os.Stat(r.asidePath("third_party/p/CHANGELOG.md")); !os.IsNotExist(err) {
		t.Fatal("ignored file was moved aside")
	}
}

func TestRebaseRestoresLeftoverAsideAfterAbort(t *testing.T) {
	r := newGitRepo(t)
	r.enter()
	aside := r.asidePath("notes.txt")
	if err := os.MkdirAll(filepath.Dir(aside), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(aside, []byte("kept\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := rebaseOnto("main"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(r.dir, "notes.txt"))
	if err != nil || string(body) != "kept\n" {
		t.Fatalf("restored %q err=%v", body, err)
	}
	if _, err := os.Stat(aside); !os.IsNotExist(err) {
		t.Fatal("aside copy remained")
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
	err := rebaseOnto("main")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err=%v", err)
	}
	body, err := os.ReadFile(filepath.Join(r.dir, "kept"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "from-main\n" {
		t.Fatalf("file = %q", body)
	}
	kept, err := os.ReadFile(aside)
	if err != nil || string(kept) != "local\n" {
		t.Fatalf("aside = %q err=%v", kept, err)
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
	if err := syncLogic("", false); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err=%v", err)
	}
	body, err = os.ReadFile(filepath.Join(r.dir, "kept"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "from-main\n" {
		t.Fatalf("file = %q", body)
	}
	aside, err := os.ReadFile(r.asidePath("kept"))
	if err != nil || string(aside) != "local\n" {
		t.Fatalf("aside = %q err=%v", aside, err)
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
	cmd := newPolishCommand()
	cmd.SetContext(context.Background())
	buf := captureText(t)
	err := polishRun(cmd)
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "left in place") {
		t.Fatalf("out=%q", buf.String())
	}
	body, err := os.ReadFile(filepath.Join(r.dir, "third_party", "p", "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "regenerated\n" {
		t.Fatalf("file = %q", body)
	}
	if _, statErr := os.Stat(r.asidePath("third_party/p/CHANGELOG.md")); !os.IsNotExist(statErr) {
		t.Fatal("ignored file was moved aside")
	}
}

func TestAcceptRefusesUntrackedCollisionBeforePrep(t *testing.T) {
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
	cmd := newAcceptCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := cmd.Flags().Set("dirty", "carry"); err != nil {
		t.Fatal(err)
	}
	buf := captureText(t)
	err := acceptRun(cmd, []string{"B"}, spec)
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "would be overwritten") || strings.Contains(buf.String(), "recovery backup was kept") {
		t.Fatalf("out=%q", buf.String())
	}
	body, err := os.ReadFile(filepath.Join(r.dir, "kept"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "local\n" {
		t.Fatalf("file = %q", body)
	}
	if r.git("rev-parse", "--abbrev-ref", "HEAD") != "B" || r.git("stash", "list") != "" {
		t.Fatalf("head %s stash %q", r.git("rev-parse", "--abbrev-ref", "HEAD"), r.git("stash", "list"))
	}
	if refs := r.gitAllow("for-each-ref", "refs/elegant-git/recovery"); refs != "" {
		t.Fatalf("recovery %s", refs)
	}
}
