package work

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
)

type pruneOnFetchRunner struct {
	inner   *git.MemoryRunner
	prune   string
	fetched bool
}

func (r *pruneOnFetchRunner) Verbose(args ...string) error {
	if len(args) > 0 && args[0] == "fetch" {
		r.fetched = true
		delete(r.inner.Outputs, "rev-parse --verify --quiet "+r.prune)
		delete(r.inner.Outputs, "for-each-ref refs/remotes/"+strings.TrimPrefix(r.prune, "origin/"))
	}
	return r.inner.Verbose(args...)
}

func (r *pruneOnFetchRunner) VerboseOp(processor func(string), args ...string) error {
	return r.inner.VerboseOp(processor, args...)
}

func (r *pruneOnFetchRunner) VerboseOpLines(lineFn func(string), args ...string) error {
	return r.inner.VerboseOpLines(lineFn, args...)
}

func (r *pruneOnFetchRunner) StreamLines(lineFn func(string), args ...string) error {
	return r.inner.StreamLines(lineFn, args...)
}

func (r *pruneOnFetchRunner) Output(args ...string) (string, error) {
	return r.inner.Output(args...)
}

func (r *pruneOnFetchRunner) OutputOK(args ...string) string {
	return r.inner.OutputOK(args...)
}

func setupSyncTest(t *testing.T, currentBranch, recordedSource, prunedRef string) (*git.MemoryRunner, git.Runner) {
	t.Helper()
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(gitDir, "elegant-git", "state.json"))

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	m.Repo.CurrentBranch = currentBranch
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["rev-parse --verify --quiet main"] = "abc"
	m.Outputs["rev-parse --verify --quiet origin/main"] = "abc"
	m.Outputs["rev-parse --verify --quiet "+prunedRef] = "abc"
	m.Outputs["rev-parse --verify --quiet --abbrev-ref --branches=refs/heads "+currentBranch] = currentBranch
	m.FailOn["rev-parse --abbrev-ref "+strings.TrimPrefix(prunedRef, "origin/")+"@{upstream}"] = fmt.Errorf("no upstream")
	m.FailOn["rev-parse --verify --quiet origin/"+strings.TrimPrefix(prunedRef, "origin/")] = fmt.Errorf("unknown")
	m.FailOn["for-each-ref refs/remotes/"+strings.TrimPrefix(prunedRef, "origin/")] = fmt.Errorf("unknown")

	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{currentBranch: recordedSource},
	}); err != nil {
		t.Fatal(err)
	}

	runner := &pruneOnFetchRunner{inner: m, prune: prunedRef}
	git.Use(runner)
	return m, runner
}

func TestSyncLogicResolvesSourceAfterFetch(t *testing.T) {
	const (
		currentBranch  = "work"
		recordedSource = "origin/feat/dzdb-3858-integrate-identity-service"
	)
	m, runner := setupSyncTest(t, currentBranch, recordedSource, recordedSource)

	if err := syncLogic(""); err != nil {
		t.Fatalf("syncLogic: %v", err)
	}
	if !runner.(*pruneOnFetchRunner).fetched {
		t.Fatal("expected fetch before resolving rebase target")
	}

	var fetchIdx, rebaseIdx int
	for i, call := range m.Calls {
		if len(call.Args) > 0 && call.Args[0] == "fetch" {
			fetchIdx = i
		}
		if len(call.Args) > 1 && call.Args[0] == "rebase" && call.Args[1] == "origin/main" {
			rebaseIdx = i
		}
	}
	if fetchIdx == 0 || rebaseIdx == 0 || fetchIdx >= rebaseIdx {
		t.Fatalf("expected fetch before rebase onto origin/main, calls: %+v", m.Calls)
	}
}
