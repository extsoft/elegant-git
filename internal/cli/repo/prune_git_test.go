package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/recovery"
)

func TestRealGitPruneDropsSelectedStashAfterInsertion(t *testing.T) {
	r := newPruneRepo(t)
	owned := r.stash("owned", "owned-body\n")
	journal := r.saveRecovery(t, owned)
	selected := r.stash("selected-ordinary", "selected-body\n")
	var inserted string
	p := newPrunePrompter(prunePickResult{values: []string{"stash@{0}"}})
	p.beforeDrop = func() {
		inserted = r.stash("unrelated-external", "external-body\n")
	}
	if err := runPrune(p, "stashes"); err != nil {
		t.Fatal(err)
	}
	if r.hasStash(selected) || !r.hasStash(inserted) || !r.hasStash(owned) {
		t.Fatalf("selected %v inserted %v owned %v\n%s", r.hasStash(selected), r.hasStash(inserted), r.hasStash(owned), r.stashList())
	}
	if !strings.Contains(r.patch(inserted), "external-body") || !strings.Contains(r.patch(owned), "owned-body") {
		t.Fatalf("patches changed\n%s", r.stashList())
	}
	if got := r.journal(t); got != journal {
		t.Fatalf("journal changed\n%s", got)
	}
}

func TestRealGitPruneRefusesWhenSelectedStashDisappears(t *testing.T) {
	r := newPruneRepo(t)
	keeper := r.stash("keeper", "keeper-body\n")
	selected := r.stash("selected-ordinary", "selected-body\n")
	p := newPrunePrompter(prunePickResult{values: []string{"stash@{0}"}})
	p.beforeDrop = func() {
		r.git("stash", "drop", "stash@{0}")
	}
	err := runPrune(p, "stashes")
	if err == nil || !strings.Contains(err.Error(), selected) || !strings.Contains(err.Error(), "nothing was dropped") {
		t.Fatalf("err=%v", err)
	}
	if r.hasStash(selected) || !r.hasStash(keeper) || !strings.Contains(r.patch(keeper), "keeper-body") {
		t.Fatalf("stashes\n%s", r.stashList())
	}
}

func TestRealGitPruneRefusesWhenSelectedStashIsReplaced(t *testing.T) {
	r := newPruneRepo(t)
	selected := r.stash("selected-ordinary", "selected-body\n")
	var replaced string
	p := newPrunePrompter(prunePickResult{values: []string{"stash@{0}"}})
	p.beforeDrop = func() {
		r.git("stash", "drop", "stash@{0}")
		replaced = r.stash("replaced", "replaced-body\n")
	}
	err := runPrune(p, "stashes")
	if err == nil || !strings.Contains(err.Error(), selected) || !strings.Contains(err.Error(), "nothing was dropped") {
		t.Fatalf("err=%v", err)
	}
	if r.hasStash(selected) || !r.hasStash(replaced) || !strings.Contains(r.patch(replaced), "replaced-body") {
		t.Fatalf("stashes\n%s", r.stashList())
	}
}

func TestRealGitPruneDropsShiftedSelections(t *testing.T) {
	r := newPruneRepo(t)
	oldest := r.stash("oldest", "oldest-body\n")
	keeper := r.stash("keeper", "keeper-body\n")
	newest := r.stash("newest", "newest-body\n")
	var inserted string
	p := newPrunePrompter(prunePickResult{values: []string{"stash@{0}", "stash@{2}"}})
	p.beforeDrop = func() {
		inserted = r.stash("inserted", "inserted-body\n")
	}
	if err := runPrune(p, "stashes"); err != nil {
		t.Fatal(err)
	}
	if r.hasStash(newest) || r.hasStash(oldest) || !r.hasStash(keeper) || !r.hasStash(inserted) {
		t.Fatalf("stashes\n%s", r.stashList())
	}
	if !strings.Contains(r.patch(keeper), "keeper-body") || !strings.Contains(r.patch(inserted), "inserted-body") {
		t.Fatalf("patches\n%s", r.stashList())
	}
}

func TestRealGitPruneDropsStashWhoseSubjectContainsATab(t *testing.T) {
	r := newPruneRepo(t)
	plain := r.stash("plain", "plain-body\n")
	tabbed := r.stash("weird\tname \"quoted\"", "tab-body\n")
	p := newPrunePrompter(prunePickResult{values: []string{"stash@{0}"}})
	if err := runPrune(p, "stashes"); err != nil {
		t.Fatal(err)
	}
	if r.hasStash(tabbed) || !r.hasStash(plain) || !strings.Contains(r.patch(plain), "plain-body") {
		t.Fatalf("stashes\n%s", r.stashList())
	}
}

func TestRealGitPruneKeepsRecoveryStashWhenItsNumberChanges(t *testing.T) {
	r := newPruneRepo(t)
	owned := r.stash("owned", "owned-body\n")
	journal := r.saveRecovery(t, owned)
	selected := r.stash("selected-ordinary", "selected-body\n")
	p := newPrunePrompter(prunePickResult{values: []string{"stash@{0}"}})
	p.beforeDrop = func() {
		r.git("stash", "drop", "stash@{0}")
	}
	err := runPrune(p, "stashes")
	if err == nil || !strings.Contains(err.Error(), "nothing was dropped") {
		t.Fatalf("err=%v", err)
	}
	if !r.hasStash(owned) || r.hasStash(selected) || !strings.Contains(r.patch(owned), "owned-body") {
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
