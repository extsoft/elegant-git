package work

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
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

func TestSyncCommandValidArgsFunction(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Repo.CurrentBranch = "feat"
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["rev-parse --verify --quiet main"] = "abc"
	m.Outputs["rev-parse --verify --quiet origin/main"] = "abc"
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "main"
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/main"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	cmd := newSyncCommand()
	if cmd.ValidArgsFunction == nil {
		t.Fatal("expected ValidArgsFunction")
	}
	out, dir := cmd.ValidArgsFunction(cmd, nil, "")
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("dir=%v", dir)
	}
	if len(out) < 2 {
		t.Fatalf("got %v", out)
	}
	if !strings.HasPrefix(out[0], "origin/main\t") || !strings.Contains(out[0], "Source branch.") {
		t.Fatalf("first completion %q", out[0])
	}
	for _, want := range []string{"main", "origin/main"} {
		found := false
		for _, line := range out {
			if strings.HasPrefix(line, want) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing %q in %v", want, out)
		}
	}
}

func TestSyncCommandHelpExplainsSource(t *testing.T) {
	var buf strings.Builder
	cliruntime.PrintCommandHelp(&buf, newSyncCommand())
	out := buf.String()
	for _, want := range []string{
		"sync [branch-name]",
		"freshest source",
		"pushed upstream",
		"interactive mode",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q:\n%s", want, out)
		}
	}
}

func TestSyncRunPromptsBranchWhenOmitted(t *testing.T) {
	const currentBranch = "feat"
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(gitDir, "elegant-git", "state.json"))

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	m.Repo.CurrentBranch = currentBranch
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["rev-parse --verify --quiet main"] = "abc"
	m.Outputs["rev-parse --verify --quiet origin/main"] = "abc"
	m.Outputs["rev-parse --verify --quiet --abbrev-ref --branches=refs/heads "+currentBranch] = currentBranch
	m.Outputs["rev-parse --abbrev-ref "+currentBranch+"@{upstream}"] = "origin/feat"
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "feat\torigin/feat\nmain\t"
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/feat\norigin/main"
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{currentBranch: "main"},
	}); err != nil {
		t.Fatal(err)
	}
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	p := &syncRunPrompter{picks: []string{"origin/main"}}
	cmd := newSyncCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), p))
	if err := syncRun(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if len(p.labels) != 1 || p.labels[0] != "Branch name" {
		t.Fatalf("labels=%v", p.labels)
	}
	if !hasGitCall(m, "rebase", "origin/main") {
		t.Fatalf("calls=%+v", m.Calls)
	}
	if n := fetchCallCount(m); n != 1 {
		t.Fatalf("fetch count=%d, calls=%+v", n, m.Calls)
	}
}

func TestSyncRunCancelDoesNotRebase(t *testing.T) {
	const currentBranch = "feat"
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(gitDir, "elegant-git", "state.json"))

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	m.Repo.CurrentBranch = currentBranch
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["rev-parse --verify --quiet main"] = "abc"
	m.Outputs["rev-parse --verify --quiet origin/main"] = "abc"
	m.Outputs["rev-parse --verify --quiet --abbrev-ref --branches=refs/heads "+currentBranch] = currentBranch
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "main"
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/main"
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{currentBranch: "main"},
	}); err != nil {
		t.Fatal(err)
	}
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	p := &syncRunPrompter{pickErr: prompt.ErrUserCancelled}
	cmd := newSyncCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), p))
	err := syncRun(cmd, nil)
	if !errors.Is(err, prompt.ErrUserCancelled) {
		t.Fatalf("err=%v", err)
	}
	if hasGitCall(m, "rebase") {
		t.Fatalf("rebased after cancel, calls=%+v", m.Calls)
	}
}

type syncRunPrompter struct {
	picks   []string
	pickErr error
	idx     int
	labels  []string
}

func (p *syncRunPrompter) String(string, string) (string, error) { return "", prompt.ErrNonInteractive }
func (p *syncRunPrompter) Confirm(string, bool) (bool, error)    { return false, nil }
func (p *syncRunPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}
func (p *syncRunPrompter) Pick(label string, _ []prompt.Choice, def string) (string, error) {
	p.labels = append(p.labels, label)
	if p.pickErr != nil {
		return "", p.pickErr
	}
	if p.idx < len(p.picks) {
		v := p.picks[p.idx]
		p.idx++
		return v, nil
	}
	if def != "" {
		return def, nil
	}
	return "", prompt.ErrNonInteractive
}
func (p *syncRunPrompter) Required(string, string) error { return nil }
func (p *syncRunPrompter) EditOrAccept(string, string) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *syncRunPrompter) Optional(string, string) (string, error) { return "", nil }
func (p *syncRunPrompter) Closed(string, []string, string, bool) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *syncRunPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}

func hasGitCall(m *git.MemoryRunner, args ...string) bool {
	for _, c := range m.Calls {
		if len(c.Args) >= len(args) {
			ok := true
			for i := range args {
				if c.Args[i] != args[i] {
					ok = false
					break
				}
			}
			if ok {
				return true
			}
		}
	}
	return false
}

func fetchCallCount(m *git.MemoryRunner) int {
	n := 0
	for _, c := range m.Calls {
		if len(c.Args) > 0 && c.Args[0] == "fetch" {
			n++
		}
	}
	return n
}

func TestSyncRunPromptsWithPickerRepoStubs(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(dir, "repo-state.json"))
	m := git.NewMemoryRunner()
	m.Repo.CurrentBranch = "feat"
	m.Outputs["rev-parse --git-dir"] = filepath.Join(dir, ".git")
	m.Outputs["rev-parse --is-inside-work-tree"] = "true"
	m.Outputs["rev-parse --show-toplevel"] = dir
	m.Outputs["rev-list --left-right --count feat...main"] = "1\t0"
	m.Repo.LocalConfig["elegant-git.repo-id"] = "abc"
	m.Outputs["rev-list --left-right --count HEAD...@{upstream}"] = "0\t2"
	m.Outputs["rev-parse --abbrev-ref feat@{upstream}"] = "origin/feat"
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["rev-parse --verify --quiet main"] = "abc"
	m.Outputs["rev-parse --verify --quiet origin/main"] = "abc"
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "feat\torigin/feat\nmain\t"
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/feat\norigin/main"
	gitDir := filepath.Join(dir, ".git")
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(gitDir, "elegant-git", "state.json"))
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{"feat": "main"},
	}); err != nil {
		t.Fatal(err)
	}
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	p := &syncRunPrompter{picks: []string{"origin/main"}}
	cmd := newSyncCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), p))
	if err := syncRun(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if len(p.labels) != 1 || p.labels[0] != "Branch name" {
		t.Fatalf("labels=%v", p.labels)
	}
	if !hasGitCall(m, "rebase", "origin/main") {
		t.Fatalf("calls=%+v", m.Calls)
	}
	if n := fetchCallCount(m); n != 1 {
		t.Fatalf("fetch count=%d, calls=%+v", n, m.Calls)
	}
}

func TestSyncLogicResolvesSourceAfterFetch(t *testing.T) {
	const (
		currentBranch  = "work"
		recordedSource = "origin/feat/dzdb-3858-integrate-identity-service"
	)
	m, runner := setupSyncTest(t, currentBranch, recordedSource, recordedSource)

	if err := syncLogic("", false); err != nil {
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
