package repo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/recovery"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func setupPrune(t *testing.T) *git.MemoryRunner {
	t.Helper()
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(gitDir, "elegant-git", "state.json"))
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	m.Outputs["rev-parse --git-common-dir"] = gitDir
	m.FailOn["rev-parse --abbrev-ref main@{upstream}"] = fmt.Errorf("no upstream")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	pruneNowUnix = func() int64 { return pruneTestNow }
	t.Cleanup(func() { pruneNowUnix = func() int64 { return time.Now().Unix() } })
	return m
}

const pruneTestNow = int64(1_760_000_000)

func pruneRow(name string, agoSec int64, date, extra string) string {
	return fmt.Sprintf("%s\t%d\t%s\t%s", name, pruneTestNow-agoSec, date, extra)
}

func stashLine(oid, ref string, agoSec int64, date, subject string) string {
	return fmt.Sprintf("%s\t%s\t%d\t%s\t%s", oid, ref, pruneTestNow-agoSec, date, subject)
}

const stashDropListKey = "stash list --format=%H%x00%gd%x00%gs"

type stashRow struct {
	oid, ref, subject string
	ago               int64
	date              string
}

func setStashes(m *git.MemoryRunner, items ...stashRow) {
	var show, drop []string
	for _, it := range items {
		show = append(show, stashLine(it.oid, it.ref, it.ago, it.date, it.subject))
		drop = append(drop, it.oid+"\x00"+it.ref+"\x00"+it.subject)
	}
	m.Outputs["stash list --format="+pruneStashFormat] = strings.Join(show, "\n")
	m.Outputs[stashDropListKey] = strings.Join(drop, "\n")
}

func mergedIntoOutput(m *git.MemoryRunner, ref, prefix string, names ...string) {
	m.Outputs["for-each-ref --merged="+ref+" --format=%(refname:short) "+prefix] = strings.Join(names, "\n")
}

func savePruneState(t *testing.T, m *git.MemoryRunner, s *memrepo.State) {
	t.Helper()
	if err := memrepo.Save(m.Outputs["rev-parse --git-dir"], s); err != nil {
		t.Fatal(err)
	}
}

const (
	day  = int64(24 * 60 * 60)
	week = 7 * day
	year = 365 * day
)

func runPrune(p prompt.Prompter, args ...string) error {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetContext(prompt.WithPrompter(context.Background(), p))
	return pruneRun(cmd, args)
}

func gitCalls(m *git.MemoryRunner) []string {
	out := make([]string, len(m.Calls))
	for i, c := range m.Calls {
		out[i] = strings.Join(c.Args, " ")
	}
	return out
}

func callIndex(calls []string, want string) int {
	for i, c := range calls {
		if c == want {
			return i
		}
	}
	return -1
}

type prunePickResult struct {
	values []string
	err    error
}

type prunePickCall struct {
	label    string
	selected []string
	choices  []prompt.Choice
}

type prunePrompter struct {
	inner      prompt.Prompter
	results    []prunePickResult
	calls      []prunePickCall
	decline    bool
	questions  []string
	beforeDrop func()
}

func newPrunePrompter(results ...prunePickResult) *prunePrompter {
	return &prunePrompter{inner: prompt.NewNonInteractive(), results: results}
}

func (p *prunePrompter) PickMany(label string, choices []prompt.Choice, selected []string) ([]string, error) {
	p.calls = append(p.calls, prunePickCall{
		label:    label,
		selected: append([]string(nil), selected...),
		choices:  append([]prompt.Choice(nil), choices...),
	})
	if len(p.results) == 0 {
		return nil, fmt.Errorf("unexpected PickMany %q", label)
	}
	r := p.results[0]
	p.results = p.results[1:]
	if r.err != nil {
		return nil, r.err
	}
	return r.values, nil
}

func (p *prunePrompter) String(q, d string) (string, error) { return p.inner.String(q, d) }
func (p *prunePrompter) Confirm(q string, yes bool) (bool, error) {
	if yes {
		return false, fmt.Errorf("confirm %q defaults to yes", q)
	}
	p.questions = append(p.questions, q)
	if p.beforeDrop != nil {
		p.beforeDrop()
	}
	return !p.decline, nil
}
func (p *prunePrompter) Choose(q string, opts []string) (int, error) {
	return p.inner.Choose(q, opts)
}
func (p *prunePrompter) Pick(label string, choices []prompt.Choice, def string) (string, error) {
	return p.inner.Pick(label, choices, def)
}
func (p *prunePrompter) Required(label, current string) error {
	return p.inner.Required(label, current)
}
func (p *prunePrompter) EditOrAccept(label, suggested string) (string, error) {
	return p.inner.EditOrAccept(label, suggested)
}
func (p *prunePrompter) Optional(label, suggested string) (string, error) {
	return p.inner.Optional(label, suggested)
}
func (p *prunePrompter) Closed(q string, opts []string, def string, required bool) (string, error) {
	return p.inner.Closed(q, opts, def, required)
}
func (p *prunePrompter) BatchChoice(q, def string) (prompt.BatchDecision, error) {
	return p.inner.BatchChoice(q, def)
}

func TestPruneNonInteractiveDefaultLocalOnly(t *testing.T) {
	m := setupPrune(t)
	m.Outputs["for-each-ref --sort=committerdate --format "+pruneRefFormat+" refs/heads"] = strings.Join([]string{
		pruneRow("old", 6*year, "2019-10-10", "origin/old"),
		pruneRow("gone", 5*year, "2020-10-10", ""),
		pruneRow("feat", 2*year, "2023-10-10", "origin/feat"),
		pruneRow("main", day, "2025-10-08", "origin/main"),
	}, "\n")
	m.Repo.LocalConfig["branch.gone.merge"] = "refs/heads/gone"
	m.FailOn["rev-parse --abbrev-ref gone@{upstream}"] = fmt.Errorf("no upstream")
	mergedIntoOutput(m, "main", "refs/heads", "old", "main")

	if err := runPrune(prompt.NewNonInteractive()); err != nil {
		t.Fatal(err)
	}
	calls := gitCalls(m)
	oldAt := callIndex(calls, "branch --delete --force old")
	goneAt := callIndex(calls, "branch --delete --force gone")
	if oldAt < 0 || goneAt < 0 || oldAt > goneAt {
		t.Fatalf("delete order: %v", calls)
	}
	if callIndex(calls, "branch --delete --force feat") >= 0 || callIndex(calls, "branch --delete --force main") >= 0 {
		t.Fatalf("deleted a kept branch: %v", calls)
	}
	for _, c := range calls {
		if strings.HasPrefix(c, "push ") || strings.HasPrefix(c, "stash ") || strings.HasPrefix(c, "worktree ") {
			t.Fatalf("unexpected %q", c)
		}
	}
}

func TestPruneHelpNamesDirtyDestination(t *testing.T) {
	cmd := newPruneCommand()
	if cmd.Flags().Lookup("dirty") == nil || cmd.Flags().Lookup("stash-name") == nil {
		t.Fatal("missing dirty flags")
	}
	for _, want := range []string{"default development branch", "--dirty", "carry", "stash", "discard", "cancel"} {
		if !strings.Contains(cmd.Long, want) {
			t.Fatalf("help missing %q", want)
		}
	}
}

func TestPrunePhaseListCancelDoesNotCheckout(t *testing.T) {
	m := setupPrune(t)
	m.Repo.CurrentBranch = "feature"
	p := newPrunePrompter(prunePickResult{err: prompt.ErrUserCancelled})
	err := runPrune(p)
	if !errors.Is(err, prompt.ErrUserCancelled) {
		t.Fatalf("got %v", err)
	}
	for _, c := range gitCalls(m) {
		if c == "checkout main" || strings.HasPrefix(c, "stash ") || strings.HasPrefix(c, "fetch ") || c == "rebase" {
			t.Fatalf("prepared before cancel: %v", gitCalls(m))
		}
	}
}

func TestPruneEmptyPhaseListDoesNotCheckout(t *testing.T) {
	m := setupPrune(t)
	if err := runPrune(newPrunePrompter(prunePickResult{})); err != nil {
		t.Fatal(err)
	}
	if callIndex(gitCalls(m), "checkout main") >= 0 {
		t.Fatalf("checked out: %v", gitCalls(m))
	}
}

func TestPruneNonInteractiveStashesLists(t *testing.T) {
	for _, args := range [][]string{{"stashes"}, {"local", "stashes"}} {
		t.Run(strings.Join(args, ","), func(t *testing.T) {
			m := setupPrune(t)
			buf := &bytes.Buffer{}
			text.SetOutput(buf)
			t.Cleanup(func() { text.SetOutput(os.Stdout) })
			if err := runPrune(prompt.NewNonInteractive(), args...); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(buf.String(), "There are no stashes.") {
				t.Fatalf("output:\n%s", buf.String())
			}
			for _, c := range gitCalls(m) {
				if strings.HasPrefix(c, "stash drop") {
					t.Fatalf("dropped from an empty list: %v", gitCalls(m))
				}
			}
			if callIndex(gitCalls(m), "checkout main") < 0 {
				t.Fatal("did not reach the phase")
			}
		})
	}
}

func TestPruneRemotePushDeleteOnce(t *testing.T) {
	m := setupPrune(t)
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["for-each-ref --sort=committerdate --format "+pruneRefFormat+" refs/remotes/origin"] = strings.Join([]string{
		pruneRow("origin/HEAD", 6*year, "2019-10-10", ""),
		pruneRow("origin/old", 6*year, "2019-10-10", ""),
		pruneRow("origin/main", 2*year, "2023-10-10", ""),
		pruneRow("origin/kept", 2*year, "2023-10-10", ""),
		pruneRow("origin/newer", year, "2024-10-09", ""),
	}, "\n")
	mergedIntoOutput(m, "main", "refs/remotes/origin", "origin/old", "origin/main", "origin/newer")

	if err := runPrune(prompt.NewNonInteractive(), "remote"); err != nil {
		t.Fatal(err)
	}
	calls := gitCalls(m)
	want := "push origin --delete old newer"
	if callIndex(calls, want) < 0 {
		t.Fatalf("calls: %v", calls)
	}
	n := 0
	for _, c := range calls {
		if strings.HasPrefix(c, "push ") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("pushes=%d calls=%v", n, calls)
	}
	if callIndex(calls, "for-each-ref --sort=committerdate --format "+pruneRefFormat+" refs/remotes/origin") < 0 {
		t.Fatalf("missing sort: %v", calls)
	}
}

func TestPruneStashesDropDescending(t *testing.T) {
	m := setupPrune(t)
	newest := strings.Repeat("a", 40)
	mid := strings.Repeat("b", 40)
	oldest := strings.Repeat("c", 40)
	setStashes(m,
		stashRow{newest, "stash@{0}", "wip new", 2 * week, "2025-09-25"},
		stashRow{mid, "stash@{1}", "wip mid", 30 * week, "2025-03-13"},
		stashRow{oldest, "stash@{2}", "wip old", 6 * year, "2019-10-10"},
	)
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	p := newPrunePrompter(prunePickResult{values: []string{"stash@{0}", "stash@{2}"}})
	if err := runPrune(p, "stashes"); err != nil {
		t.Fatal(err)
	}
	if len(p.calls) != 1 || p.calls[0].label != "Stashes to drop" {
		t.Fatalf("picker: %+v", p.calls)
	}
	if strings.Join(p.calls[0].selected, ",") != "stash@{2},stash@{1},stash@{0}" {
		t.Fatalf("defaults: %v", p.calls[0].selected)
	}
	for _, c := range p.calls[0].choices {
		if strings.Contains(c.Value, "\n") || !strings.HasPrefix(c.Value, "stash@{") {
			t.Fatalf("picker value %q", c.Value)
		}
	}
	old := p.calls[0].choices[0].Display
	middle := p.calls[0].choices[1].Display
	newer := p.calls[0].choices[2].Display
	if old != "[ 6 y ago]  2019-10-10  stash@{2}" || p.calls[0].choices[0].Description != "wip old" ||
		middle != "[30 w ago]  2025-03-13  stash@{1}" || newer != "[14 d ago]  2025-09-25  stash@{0}" {
		t.Fatalf("choices: %+v", p.calls[0].choices)
	}
	if strings.Join(p.questions, "|") != "Drop 2 stashes?" {
		t.Fatalf("questions: %v", p.questions)
	}
	calls := gitCalls(m)
	first := callIndex(calls, "stash drop stash@{0}")
	second := callIndex(calls, "stash drop stash@{2}")
	if first < 0 || second < 0 || first > second || callIndex(calls, "stash drop stash@{1}") >= 0 {
		t.Fatalf("drops: %v", calls)
	}
	if !strings.Contains(buf.String(), "Dropped 2 stashes.") {
		t.Fatalf("output:\n%s", buf.String())
	}
}

func TestPruneRefusesWhenSelectionIsTheWholeList(t *testing.T) {
	m := setupPrune(t)
	tip := strings.Repeat("a", 40)
	older := strings.Repeat("b", 40)
	setStashes(m,
		stashRow{tip, "stash@{0}", "wip new", 2 * week, "2025-09-25"},
		stashRow{older, "stash@{1}", "wip mid", 30 * week, "2025-03-13"},
	)
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	calls := gitCalls(m)
	if callIndex(calls, "stash drop stash@{1}") < 0 || callIndex(calls, "stash drop stash@{0}") < 0 {
		t.Fatalf("drops: %v", calls)
	}
	if !strings.Contains(buf.String(), "Dropped 2 stashes.") {
		t.Fatalf("output:\n%s", buf.String())
	}
}

func TestPruneHidesRecoveryStash(t *testing.T) {
	m := setupPrune(t)
	gitDir := m.Outputs["rev-parse --git-dir"]
	if err := os.MkdirAll(filepath.Join(gitDir, "elegant-git"), 0o755); err != nil {
		t.Fatal(err)
	}
	recorded := "eg work.sync auto-stash abc"
	body := []byte(`{"work.sync":{"stash":"` + recorded + `"}}` + "\n")
	if err := os.WriteFile(filepath.Join(gitDir, "elegant-git", "commands.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	recordedOID := strings.Repeat("a", 40)
	orphan := strings.Repeat("b", 40)
	wip := strings.Repeat("c", 40)
	setStashes(m,
		stashRow{recordedOID, "stash@{0}", "On main: " + recorded, day, "2025-10-08"},
		stashRow{orphan, "stash@{1}", "On main: eg work.sync auto-stash orphan", 2 * week, "2025-09-25"},
		stashRow{wip, "stash@{2}", "wip", 3 * week, "2025-09-18"},
	)
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	calls := gitCalls(m)
	if callIndex(calls, "stash drop stash@{0}") >= 0 || callIndex(calls, "stash drop stash@{1}") < 0 || callIndex(calls, "stash drop stash@{2}") < 0 {
		t.Fatalf("drops: %v", calls)
	}
	if strings.Contains(buf.String(), recorded) {
		t.Fatalf("offered a recorded backup:\n%s", buf.String())
	}
}

func TestPruneHidesRecoveryStashAfterRenumber(t *testing.T) {
	m := setupPrune(t)
	common := m.Outputs["rev-parse --git-common-dir"]
	const oid = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := recovery.Save(common, recovery.File{Ops: []recovery.Op{{
		ID: "abc", Command: "work.sync", Phase: recovery.PhaseSuspended, StashOID: oid,
	}}}); err != nil {
		t.Fatal(err)
	}
	ordinary := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	setStashes(m,
		stashRow{ordinary, "stash@{0}", "ordinary", day, "2025-10-08"},
		stashRow{oid, "stash@{1}", "renamed-subject", 2 * week, "2025-09-25"},
	)
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	calls := gitCalls(m)
	if callIndex(calls, "stash drop stash@{0}") < 0 || callIndex(calls, "stash drop stash@{1}") >= 0 {
		t.Fatalf("drops: %v", calls)
	}
	if strings.Contains(buf.String(), oid) {
		t.Fatalf("offered a journaled backup:\n%s", buf.String())
	}
}

func TestPruneStashesRefuseWhenCommonDirIsUnknown(t *testing.T) {
	m := setupPrune(t)
	delete(m.Outputs, "rev-parse --git-common-dir")
	m.Outputs["stash list --format="+pruneStashFormat] = stashLine(strings.Repeat("a", 40), "stash@{0}", day, "2025-10-08", "wip")
	p := newPrunePrompter(prunePickResult{values: []string{"stash@{0}"}})
	if err := runPrune(p, "stashes"); err == nil {
		t.Fatal("expected missing common dir to refuse prune")
	}
	if callIndex(gitCalls(m), "stash drop stash@{0}") >= 0 {
		t.Fatal("dropped a stash without a common dir")
	}
}

func TestPruneLocalOldestFirst(t *testing.T) {
	m := setupPrune(t)
	m.Outputs["for-each-ref --sort=committerdate --format "+pruneRefFormat+" refs/heads"] = strings.Join([]string{
		pruneRow("old", 6*year, "2019-10-10", "origin/old"),
		pruneRow("newer-"+strings.Repeat("x", 40), 3*week, "2025-09-18", ""),
	}, "\n")
	newer := "newer-" + strings.Repeat("x", 40)
	mergedIntoOutput(m, "main", "refs/heads", "old", newer)
	p := newPrunePrompter(prunePickResult{values: []string{"old", newer}})
	if err := runPrune(p, "local"); err != nil {
		t.Fatal(err)
	}
	if len(p.calls) != 1 {
		t.Fatalf("calls: %+v", p.calls)
	}
	if strings.Join(p.calls[0].selected, ",") != "old,"+newer {
		t.Fatalf("selected: %v", p.calls[0].selected)
	}
	first := p.calls[0].choices[0]
	if first.Display != "[ 6 y ago]  2019-10-10  old" || first.Description != "merged into main" {
		t.Fatalf("first: %+v", first)
	}
	second := p.calls[0].choices[1]
	if second.Value != newer {
		t.Fatalf("value trimmed: %q", second.Value)
	}
	if second.Display != "[21 d ago]  2025-09-18  newer-"+strings.Repeat("x", 31)+"..." || second.Description != "merged into main" {
		t.Fatalf("second: %+v", second)
	}
	calls := gitCalls(m)
	if callIndex(calls, "for-each-ref --sort=committerdate --format "+pruneRefFormat+" refs/heads") < 0 {
		t.Fatalf("missing sort: %v", calls)
	}
	oldAt := callIndex(calls, "branch --delete --force old")
	newerAt := callIndex(calls, "branch --delete --force "+newer)
	if oldAt < 0 || newerAt < 0 || oldAt > newerAt {
		t.Fatalf("order: %v", calls)
	}
}

func TestPruneEscSkipsPhase(t *testing.T) {
	m := setupPrune(t)
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["for-each-ref --sort=committerdate --format "+pruneRefFormat+" refs/heads"] = pruneRow("old", 6*year, "2019-10-10", "")
	m.Outputs["for-each-ref --sort=committerdate --format "+pruneRefFormat+" refs/remotes/origin"] = pruneRow("origin/old", 6*year, "2019-10-10", "")
	mergedIntoOutput(m, "main", "refs/heads", "old")
	mergedIntoOutput(m, "main", "refs/remotes/origin", "origin/old")
	p := newPrunePrompter(
		prunePickResult{err: prompt.ErrUserCancelled},
		prunePickResult{values: []string{"origin/old"}},
	)
	if err := runPrune(p, "local", "remote"); err != nil {
		t.Fatal(err)
	}
	calls := gitCalls(m)
	if callIndex(calls, "branch --delete --force old") >= 0 {
		t.Fatalf("deleted skipped phase: %v", calls)
	}
	if callIndex(calls, "push origin --delete old") < 0 {
		t.Fatalf("missing remote delete: %v", calls)
	}
}

func TestPruneInteractiveAsksPhases(t *testing.T) {
	m := setupPrune(t)
	m.Outputs["worktree list --porcelain"] = "worktree /gone/old\nHEAD b\nprunable gitdir file points to non-existent location\n"
	p := newPrunePrompter(
		prunePickResult{values: []string{"worktrees"}},
		prunePickResult{values: []string{"/gone/old"}},
	)
	if err := runPrune(p); err != nil {
		t.Fatal(err)
	}
	if len(p.calls) != 2 || p.calls[0].label != "What to clean up" || p.calls[1].label != "Worktrees to prune" {
		t.Fatalf("calls: %+v", p.calls)
	}
	if strings.Join(p.calls[0].selected, ",") != "local,remote,stashes,worktrees" {
		t.Fatalf("phase defaults: %v", p.calls[0].selected)
	}
	if callIndex(gitCalls(m), "worktree prune") < 0 {
		t.Fatalf("calls: %v", gitCalls(m))
	}
}

func TestPrunePhasesRunInFixedOrder(t *testing.T) {
	m := setupPrune(t)
	m.Outputs["for-each-ref --sort=committerdate --format "+pruneRefFormat+" refs/heads"] = pruneRow("old", 6*year, "2019-10-10", "")
	mergedIntoOutput(m, "main", "refs/heads", "old")
	m.Outputs["worktree list --porcelain"] = "worktree /gone/old\nHEAD b\nprunable gitdir file points to non-existent location\n"
	if err := runPrune(prompt.NewNonInteractive(), "worktrees", "local"); err != nil {
		t.Fatal(err)
	}
	calls := gitCalls(m)
	localAt := callIndex(calls, "branch --delete --force old")
	treeAt := callIndex(calls, "worktree prune")
	if localAt < 0 || treeAt < 0 || localAt > treeAt {
		t.Fatalf("order: %v", calls)
	}
}

func TestPrunePrintsPhaseHeaders(t *testing.T) {
	setupPrune(t)
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	if err := runPrune(prompt.NewNonInteractive(), "worktrees", "local"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	local := strings.Index(out, "==>> Pruning local branches merged into main or a protected branch, or whose upstream is gone...\n")
	trees := strings.Index(out, "==>> Pruning worktrees whose directory no longer exists...\n")
	if local < 0 || trees < 0 || local > trees {
		t.Fatalf("headers:\n%s", out)
	}
	for _, want := range []string{
		"No local branches are merged into main or a protected branch, or have a gone upstream.\n",
		"No worktrees are missing their directory.\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
}

func TestPruneConfirmsBeforeApplying(t *testing.T) {
	m := setupPrune(t)
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["for-each-ref --sort=committerdate --format "+pruneRefFormat+" refs/remotes/origin"] = strings.Join([]string{
		pruneRow("origin/a", 2*year, "2023-10-10", ""),
		pruneRow("origin/b", year, "2024-10-09", ""),
	}, "\n")
	mergedIntoOutput(m, "main", "refs/remotes/origin", "origin/a", "origin/b")
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	p := newPrunePrompter(prunePickResult{values: []string{"origin/a", "origin/b"}})
	if err := runPrune(p, "remote"); err != nil {
		t.Fatal(err)
	}
	if strings.Join(p.questions, "|") != "Delete 2 branches on origin?" {
		t.Fatalf("questions: %v", p.questions)
	}
	if callIndex(gitCalls(m), "push origin --delete a b") < 0 {
		t.Fatalf("calls: %v", gitCalls(m))
	}
	if !strings.Contains(buf.String(), "Deleted 2 branches on origin.\n") {
		t.Fatalf("summary:\n%s", buf.String())
	}
}

func TestPruneDeclinedConfirmDeletesNothing(t *testing.T) {
	m := setupPrune(t)
	oid := strings.Repeat("a", 40)
	setStashes(m, stashRow{oid, "stash@{0}", "wip", day, "2025-10-08"})
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	p := newPrunePrompter(prunePickResult{values: []string{"stash@{0}"}})
	p.decline = true
	if err := runPrune(p, "stashes"); err != nil {
		t.Fatal(err)
	}
	for _, c := range gitCalls(m) {
		if strings.HasPrefix(c, "stash drop") {
			t.Fatalf("dropped: %v", gitCalls(m))
		}
	}
	if !strings.Contains(buf.String(), "Nothing dropped.") {
		t.Fatalf("output:\n%s", buf.String())
	}
}

func TestPruneEmptySelectionSkipsConfirm(t *testing.T) {
	m := setupPrune(t)
	m.Outputs["for-each-ref --sort=committerdate --format "+pruneRefFormat+" refs/heads"] = pruneRow("old", year, "2024-10-09", "")
	mergedIntoOutput(m, "main", "refs/heads", "old")
	p := newPrunePrompter(prunePickResult{values: nil})
	if err := runPrune(p, "local"); err != nil {
		t.Fatal(err)
	}
	if len(p.questions) != 0 {
		t.Fatalf("questions: %v", p.questions)
	}
	if callIndex(gitCalls(m), "branch --delete --force old") >= 0 {
		t.Fatalf("calls: %v", gitCalls(m))
	}
}

func TestPruneMergedIntoProtectedBranch(t *testing.T) {
	m := setupPrune(t)
	savePruneState(t, m, &memrepo.State{
		DefaultBranch:     "main",
		ProtectedBranches: []string{"main", "master"},
	})
	m.Outputs["for-each-ref refs/remotes/origin/master"] = "origin/master"
	m.Outputs["for-each-ref --sort=committerdate --format "+pruneRefFormat+" refs/heads"] = pruneRow("only-master", year, "2024-10-09", "")
	mergedIntoOutput(m, "main", "refs/heads")
	mergedIntoOutput(m, "origin/master", "refs/heads", "only-master")
	p := newPrunePrompter(prunePickResult{values: []string{"only-master"}})
	if err := runPrune(p, "local"); err != nil {
		t.Fatal(err)
	}
	if len(p.calls) != 1 || p.calls[0].choices[0].Description != "merged into master" {
		t.Fatalf("choices: %+v", p.calls)
	}
	if callIndex(gitCalls(m), "branch --delete --force only-master") < 0 {
		t.Fatalf("calls: %v", gitCalls(m))
	}
}

func TestPruneSkipsProtectedBranchWithNoRef(t *testing.T) {
	m := setupPrune(t)
	savePruneState(t, m, &memrepo.State{
		DefaultBranch:     "main",
		ProtectedBranches: []string{"main", "ghost"},
	})
	m.FailOn["rev-parse --verify --quiet --abbrev-ref --branches=refs/heads ghost"] = fmt.Errorf("missing")
	m.Outputs["for-each-ref --sort=committerdate --format "+pruneRefFormat+" refs/heads"] = pruneRow("only-ghost", year, "2024-10-09", "")
	mergedIntoOutput(m, "main", "refs/heads")
	if err := runPrune(prompt.NewNonInteractive(), "local"); err != nil {
		t.Fatal(err)
	}
	for _, c := range gitCalls(m) {
		if strings.Contains(c, "--merged=ghost") || strings.Contains(c, "--merged=origin/ghost") {
			t.Fatalf("queried missing protected branch: %s", c)
		}
	}
	if callIndex(gitCalls(m), "branch --delete --force only-ghost") >= 0 {
		t.Fatalf("deleted: %v", gitCalls(m))
	}
}

func TestPruneSkipsDefaultBranchWhenUnprotected(t *testing.T) {
	m := setupPrune(t)
	savePruneState(t, m, &memrepo.State{
		DefaultBranch:     "develop",
		ProtectedBranches: []string{"main"},
	})
	m.FailOn["rev-parse --abbrev-ref develop@{upstream}"] = fmt.Errorf("no upstream")
	m.FailOn["rev-parse --verify --quiet --abbrev-ref --branches=refs/heads main"] = fmt.Errorf("missing")
	m.Outputs["for-each-ref --sort=committerdate --format "+pruneRefFormat+" refs/heads"] = strings.Join([]string{
		pruneRow("feat", year, "2024-10-09", ""),
		pruneRow("develop", day, "2025-10-08", ""),
	}, "\n")
	mergedIntoOutput(m, "develop", "refs/heads", "feat", "develop")
	if err := runPrune(prompt.NewNonInteractive(), "local"); err != nil {
		t.Fatal(err)
	}
	calls := gitCalls(m)
	if callIndex(calls, "branch --delete --force develop") >= 0 {
		t.Fatalf("deleted default: %v", calls)
	}
	if callIndex(calls, "branch --delete --force feat") < 0 {
		t.Fatalf("kept merged branch: %v", calls)
	}
}

func TestPruneGoneUpstreamReason(t *testing.T) {
	m := setupPrune(t)
	m.Outputs["for-each-ref --sort=committerdate --format "+pruneRefFormat+" refs/heads"] = pruneRow("gone", year, "2024-10-09", "origin/gone")
	m.Repo.LocalConfig["branch.gone.merge"] = "refs/heads/gone"
	m.FailOn["rev-parse --abbrev-ref gone@{upstream}"] = fmt.Errorf("no upstream")
	mergedIntoOutput(m, "main", "refs/heads")
	p := newPrunePrompter(prunePickResult{values: []string{"gone"}})
	if err := runPrune(p, "local"); err != nil {
		t.Fatal(err)
	}
	if len(p.calls) != 1 || p.calls[0].choices[0].Description != "upstream origin/gone is gone" {
		t.Fatalf("choices: %+v", p.calls)
	}
}

func TestPruneRemoteWithoutRemotesMessage(t *testing.T) {
	setupPrune(t)
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	if err := runPrune(prompt.NewNonInteractive(), "remote"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No remotes; skipping origin branches.\n") {
		t.Fatalf("output:\n%s", buf.String())
	}
}

func TestPruneFetchPrunesRemoteTracking(t *testing.T) {
	m := setupPrune(t)
	delete(m.FailOn, "rev-parse --abbrev-ref main@{upstream}")
	m.Outputs["rev-parse --abbrev-ref main@{upstream}"] = "origin/main"
	if err := runPrune(prompt.NewNonInteractive()); err != nil {
		t.Fatal(err)
	}
	calls := gitCalls(m)
	if callIndex(calls, "fetch --all --prune") < 0 {
		t.Fatalf("calls: %v", calls)
	}
}

func TestPruneWorktreeLocksUnselected(t *testing.T) {
	m := setupPrune(t)
	m.Outputs["worktree list --porcelain"] = strings.Join([]string{
		"worktree /repo",
		"HEAD a",
		"branch refs/heads/main",
		"",
		"worktree /gone/old",
		"HEAD b",
		"prunable gitdir file points to non-existent location",
		"",
		"worktree /gone/new",
		"HEAD c",
		"prunable gitdir file points to non-existent location",
		"",
		"worktree /locked/one",
		"HEAD d",
		"locked gone",
		"prunable gitdir file points to non-existent location",
		"",
	}, "\n")
	p := newPrunePrompter(prunePickResult{values: []string{"/gone/old"}})
	if err := runPrune(p, "worktrees"); err != nil {
		t.Fatal(err)
	}
	if len(p.calls) != 1 || strings.Join(valuesOf(p.calls[0].choices), ",") != "/gone/old,/gone/new" {
		t.Fatalf("choices: %+v", p.calls)
	}
	calls := gitCalls(m)
	lockAt := callIndex(calls, "worktree lock /gone/new")
	pruneAt := callIndex(calls, "worktree prune")
	unlockAt := callIndex(calls, "worktree unlock /gone/new")
	if lockAt < 0 || pruneAt < 0 || unlockAt < 0 || !(lockAt < pruneAt && pruneAt < unlockAt) {
		t.Fatalf("calls: %v", calls)
	}
	if callIndex(calls, "worktree lock /locked/one") >= 0 || callIndex(calls, "worktree lock /repo") >= 0 {
		t.Fatalf("locked a non-candidate: %v", calls)
	}
}

func TestPruneCommandRejectsUnknownPhase(t *testing.T) {
	cmd := newPruneCommand()
	if err := cmd.Args(cmd, []string{"nope"}); err == nil {
		t.Fatal("expected error")
	}
	if err := cmd.Args(cmd, []string{"tracking"}); err == nil {
		t.Fatal("expected error for tracking")
	}
	if err := cmd.Args(cmd, []string{"worktrees", "local"}); err != nil {
		t.Fatal(err)
	}
}

func TestPruneProgressSkippedWithoutTerminal(t *testing.T) {
	if p := newPruneProgress(&bytes.Buffer{}, "Remote branches"); p != nil {
		t.Fatalf("got %+v", p)
	}
	var p *pruneProgress
	p.setTotal(3)
	p.tick("origin/a")
	p.close()
}

func TestPruneProgressLineAndMinimumDuration(t *testing.T) {
	var slept time.Duration
	pruneSleep = func(d time.Duration) { slept = d }
	t.Cleanup(func() { pruneSleep = time.Sleep })

	out := &bytes.Buffer{}
	p := &pruneProgress{w: out, label: "Remote branches", started: time.Now()}
	p.setTotal(4)
	p.tick("origin/a")
	p.tick("origin/" + strings.Repeat("x", 50))
	if got := p.line(); got != "Scanning remote branches 2/4 : origin/"+strings.Repeat("x", 30)+"..." {
		t.Fatalf("line = %q", got)
	}
	p.close()
	if slept < time.Second || slept > 1500*time.Millisecond {
		t.Fatalf("slept %v", slept)
	}
	got := out.String()
	if !strings.Contains(got, "Scanning remote branches 4/4") {
		t.Fatalf("missing final count: %q", got)
	}
	if !strings.HasSuffix(got, "\r\033[2K") {
		t.Fatalf("line not erased: %q", got)
	}
}

func TestPruneProgressNoWaitAfterMinimum(t *testing.T) {
	called := false
	pruneSleep = func(time.Duration) { called = true }
	t.Cleanup(func() { pruneSleep = time.Sleep })

	p := &pruneProgress{w: &bytes.Buffer{}, label: "Local branches", started: time.Now().Add(-2 * time.Second)}
	p.close()
	if called {
		t.Fatal("slept after minimum elapsed")
	}
}

func valuesOf(choices []prompt.Choice) []string {
	out := make([]string, len(choices))
	for i, c := range choices {
		out[i] = c.Value
	}
	return out
}

func TestSplitStashSubjectTabDoesNotMoveOID(t *testing.T) {
	oid := strings.Repeat("d", 40)
	ref, _, _, subject, got, ok := splitStash(oid + "\tstash@{0}\t1700000000\t2023-11-14\tweird\tname \"quoted\"")
	if !ok || ref != "stash@{0}" || got != oid || subject != "weird\tname \"quoted\"" {
		t.Fatalf("ok=%v ref=%q oid=%q subject=%q", ok, ref, got, subject)
	}
}

func TestPruneDropFollowsSelectedOIDAfterInsertion(t *testing.T) {
	m := setupPrune(t)
	selected := strings.Repeat("a", 40)
	keeper := strings.Repeat("b", 40)
	inserted := strings.Repeat("c", 40)
	setStashes(m,
		stashRow{selected, "stash@{0}", "selected", day, "2025-10-08"},
		stashRow{keeper, "stash@{1}", "keeper", 2 * week, "2025-09-25"},
	)
	m.Outputs[stashDropListKey] = strings.Join([]string{
		inserted + "\x00stash@{0}\x00inserted",
		selected + "\x00stash@{1}\x00selected",
		keeper + "\x00stash@{2}\x00keeper",
	}, "\n")
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	p := newPrunePrompter(prunePickResult{values: []string{"stash@{0}"}})
	if err := runPrune(p, "stashes"); err != nil {
		t.Fatal(err)
	}
	calls := gitCalls(m)
	if callIndex(calls, "stash drop stash@{1}") < 0 || callIndex(calls, "stash drop stash@{0}") >= 0 || callIndex(calls, "stash drop stash@{2}") >= 0 {
		t.Fatalf("drops: %v", calls)
	}
	if strings.Contains(buf.String(), inserted) {
		t.Fatalf("output:\n%s", buf.String())
	}
}

func TestPruneRefusesWhenSelectedStashIsReplaced(t *testing.T) {
	m := setupPrune(t)
	selected := strings.Repeat("a", 40)
	setStashes(m, stashRow{selected, "stash@{0}", "selected", day, "2025-10-08"})
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	p := newPrunePrompter(prunePickResult{values: []string{"stash@{0}"}})
	p.beforeDrop = func() {
		setStashes(m, stashRow{selected, "stash@{0}", "other", day, "2025-10-08"})
	}
	if err := runPrune(p, "stashes"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "changed and was kept") || !strings.Contains(buf.String(), "Nothing dropped.") {
		t.Fatalf("output:\n%s", buf.String())
	}
	for _, c := range gitCalls(m) {
		if strings.HasPrefix(c, "stash drop") {
			t.Fatalf("dropped %q", c)
		}
	}
}

func TestPruneDropsShiftedSelectionInDescendingOrder(t *testing.T) {
	m := setupPrune(t)
	a := strings.Repeat("a", 40)
	b := strings.Repeat("b", 40)
	c := strings.Repeat("c", 40)
	d := strings.Repeat("d", 40)
	setStashes(m,
		stashRow{a, "stash@{0}", "a", day, "2025-10-08"},
		stashRow{b, "stash@{1}", "b", 2 * week, "2025-09-25"},
		stashRow{c, "stash@{2}", "c", 3 * week, "2025-09-18"},
	)
	n := 0
	git.BeforeDropStash = func() {
		n++
		if n != 1 {
			return
		}
		setStashes(m,
			stashRow{d, "stash@{0}", "d", day, "2025-10-08"},
			stashRow{a, "stash@{1}", "a", day, "2025-10-08"},
			stashRow{b, "stash@{2}", "b", 2 * week, "2025-09-25"},
		)
	}
	t.Cleanup(func() { git.BeforeDropStash = nil })
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	p := newPrunePrompter(prunePickResult{values: []string{"stash@{2}", "stash@{0}"}})
	if err := runPrune(p, "stashes"); err != nil {
		t.Fatal(err)
	}
	calls := gitCalls(m)
	first := callIndex(calls, "stash drop stash@{2}")
	second := callIndex(calls, "stash drop stash@{1}")
	if first < 0 || second < 0 || first > second || callIndex(calls, "stash drop stash@{0}") >= 0 {
		t.Fatalf("drops: %v", calls)
	}
	if strings.Contains(buf.String(), d) || !strings.Contains(buf.String(), "Dropped 2 stashes.") {
		t.Fatalf("output:\n%s", buf.String())
	}
}

func TestPruneRefusesToDropTheOnlyStash(t *testing.T) {
	m := setupPrune(t)
	selected := strings.Repeat("a", 40)
	setStashes(m, stashRow{selected, "stash@{0}", "selected", day, "2025-10-08"})
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	if callIndex(gitCalls(m), "stash drop stash@{0}") < 0 || !strings.Contains(buf.String(), "Dropped 1 stash.") {
		t.Fatalf("output:\n%s\ncalls: %v", buf.String(), gitCalls(m))
	}
}

func TestPruneStashCancelAndEmptySelectionDropNothing(t *testing.T) {
	oid := strings.Repeat("a", 40)
	for _, tc := range []struct {
		name   string
		result prunePickResult
	}{
		{name: "cancel", result: prunePickResult{err: prompt.ErrUserCancelled}},
		{name: "empty", result: prunePickResult{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := setupPrune(t)
			setStashes(m, stashRow{oid, "stash@{0}", "wip", day, "2025-10-08"})
			buf := &bytes.Buffer{}
			text.SetOutput(buf)
			t.Cleanup(func() { text.SetOutput(os.Stdout) })
			if err := runPrune(newPrunePrompter(tc.result), "stashes"); err != nil {
				t.Fatal(err)
			}
			for _, c := range gitCalls(m) {
				if strings.HasPrefix(c, "stash drop") {
					t.Fatalf("dropped: %v", gitCalls(m))
				}
			}
			if !strings.Contains(buf.String(), "Nothing dropped.") {
				t.Fatalf("output:\n%s", buf.String())
			}
		})
	}
}

func TestPruneDefaultPhaseDoesNotDropStashes(t *testing.T) {
	m := setupPrune(t)
	setStashes(m, stashRow{strings.Repeat("a", 40), "stash@{0}", "wip", day, "2025-10-08"})
	if err := runPrune(prompt.NewNonInteractive()); err != nil {
		t.Fatal(err)
	}
	for _, c := range gitCalls(m) {
		if strings.HasPrefix(c, "stash ") {
			t.Fatalf("touched a stash: %s", c)
		}
	}
}

func TestPruneSkipsMissingChangedAndAmbiguousStashes(t *testing.T) {
	m := setupPrune(t)
	gone := strings.Repeat("a", 40)
	changed := strings.Repeat("b", 40)
	dup := strings.Repeat("c", 40)
	setStashes(m,
		stashRow{gone, "stash@{0}", "gone", day, "2025-10-08"},
		stashRow{changed, "stash@{1}", "keep", 2 * week, "2025-09-25"},
		stashRow{dup, "stash@{2}", "same", 3 * week, "2025-09-18"},
		stashRow{dup, "stash@{3}", "On main: same", 4 * week, "2025-09-11"},
	)
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	p := newPrunePrompter(prunePickResult{values: []string{"stash@{0}", "stash@{1}", "stash@{2}"}})
	p.beforeDrop = func() {
		setStashes(m,
			stashRow{changed, "stash@{0}", "other", day, "2025-10-08"},
			stashRow{dup, "stash@{1}", "same", 2 * week, "2025-09-25"},
			stashRow{dup, "stash@{2}", "On main: same", 3 * week, "2025-09-18"},
		)
	}
	if err := runPrune(p, "stashes"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{gone + " is already gone.", changed + " changed and was kept.", dup + " is ambiguous and was kept.", "Nothing dropped."} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	for _, c := range gitCalls(m) {
		if strings.HasPrefix(c, "stash drop") {
			t.Fatalf("dropped: %v", gitCalls(m))
		}
	}
}

func TestPruneDropsStashSubjectWithTabAndQuotes(t *testing.T) {
	m := setupPrune(t)
	oid := strings.Repeat("d", 40)
	subject := "weird\tname \"quoted\""
	setStashes(m, stashRow{oid, "stash@{0}", subject, day, "2025-10-08"})
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	if callIndex(gitCalls(m), "stash drop stash@{0}") < 0 {
		t.Fatalf("calls: %v", gitCalls(m))
	}
}

func TestPruneStashBatchStopsAfterAGitFailure(t *testing.T) {
	m := setupPrune(t)
	a := strings.Repeat("a", 40)
	b := strings.Repeat("b", 40)
	c := strings.Repeat("c", 40)
	setStashes(m,
		stashRow{a, "stash@{0}", "a", day, "2025-10-08"},
		stashRow{b, "stash@{1}", "b", 2 * week, "2025-09-25"},
		stashRow{c, "stash@{2}", "c", 3 * week, "2025-09-18"},
	)
	m.FailOn["stash drop stash@{1}"] = errors.New("busy")
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	err := runPrune(prompt.NewNonInteractive(), "stashes")
	if err == nil || !strings.Contains(err.Error(), "stopped after dropping 1 stash") || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("err=%v", err)
	}
	calls := gitCalls(m)
	if callIndex(calls, "stash drop stash@{2}") < 0 || callIndex(calls, "stash drop stash@{1}") < 0 || callIndex(calls, "stash drop stash@{0}") >= 0 {
		t.Fatalf("drops: %v", calls)
	}
}

func TestPruneHoldsRecoveryLockWhileDropping(t *testing.T) {
	m := setupPrune(t)
	common := m.Outputs["rev-parse --git-common-dir"]
	setStashes(m, stashRow{strings.Repeat("a", 40), "stash@{0}", "wip", day, "2025-10-08"})
	release := make(chan struct{})
	blocked := false
	git.BeforeDropStash = func() {
		go func() {
			_ = recovery.WithLock(common, func() error {
				close(release)
				return nil
			})
		}()
		select {
		case <-release:
		case <-time.After(200 * time.Millisecond):
			blocked = true
		}
	}
	t.Cleanup(func() { git.BeforeDropStash = nil })
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	if !blocked {
		t.Fatal("drop did not hold the recovery lock")
	}
	select {
	case <-release:
	case <-time.After(2 * time.Second):
		t.Fatal("lock was not released")
	}
}

func TestPruneReleasesRecoveryLockAfterStashDropError(t *testing.T) {
	m := setupPrune(t)
	common := m.Outputs["rev-parse --git-common-dir"]
	setStashes(m, stashRow{strings.Repeat("a", 40), "stash@{0}", "wip", day, "2025-10-08"})
	m.FailOn["stash drop stash@{0}"] = errors.New("busy")
	err := runPrune(prompt.NewNonInteractive(), "stashes")
	if err == nil || !strings.Contains(err.Error(), "stopped before any stash was dropped") {
		t.Fatalf("err=%v", err)
	}
	if err := recovery.WithLock(common, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestPruneKeepsLinkedWorktreeAndOrphanRefStashes(t *testing.T) {
	m := setupPrune(t)
	common := m.Outputs["rev-parse --git-common-dir"]
	linked := strings.Repeat("a", 40)
	orphan := strings.Repeat("b", 40)
	ordinary := strings.Repeat("c", 40)
	dir := filepath.Join(common, "worktrees", "wt", "elegant-git")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"work.sync":{"stash":"eg work.sync auto-stash linked"}}` + "\n")
	if err := os.WriteFile(filepath.Join(dir, "commands.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	m.Outputs["for-each-ref --format=%(objectname) "+recovery.RefPrefix] = orphan
	setStashes(m,
		stashRow{linked, "stash@{0}", "On main: eg work.sync auto-stash linked", day, "2025-10-08"},
		stashRow{orphan, "stash@{1}", "orphan-ref", 2 * week, "2025-09-25"},
		stashRow{ordinary, "stash@{2}", "ordinary", 3 * week, "2025-09-18"},
	)
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	calls := gitCalls(m)
	if callIndex(calls, "stash drop stash@{2}") < 0 || callIndex(calls, "stash drop stash@{0}") >= 0 || callIndex(calls, "stash drop stash@{1}") >= 0 {
		t.Fatalf("drops: %v", calls)
	}
}

func TestPruneOffersStashAfterRecoveryIsAcknowledged(t *testing.T) {
	m := setupPrune(t)
	common := m.Outputs["rev-parse --git-common-dir"]
	oid := strings.Repeat("a", 40)
	if err := recovery.Save(common, recovery.File{Ops: []recovery.Op{{
		ID: "abc", Command: "work.sync", Phase: recovery.PhaseSuspended, StashOID: oid,
	}}}); err != nil {
		t.Fatal(err)
	}
	setStashes(m, stashRow{oid, "stash@{0}", "leftover", day, "2025-10-08"})
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	if callIndex(gitCalls(m), "stash drop stash@{0}") >= 0 {
		t.Fatal("dropped a journaled stash")
	}
	if err := recovery.Save(common, recovery.File{}); err != nil {
		t.Fatal(err)
	}
	m.Calls = nil
	if err := runPrune(prompt.NewNonInteractive(), "stashes"); err != nil {
		t.Fatal(err)
	}
	if callIndex(gitCalls(m), "stash drop stash@{0}") < 0 {
		t.Fatalf("calls: %v", gitCalls(m))
	}
}

func TestPruneRechecksProtectionBeforeDropping(t *testing.T) {
	m := setupPrune(t)
	common := m.Outputs["rev-parse --git-common-dir"]
	oid := strings.Repeat("a", 40)
	setStashes(m, stashRow{oid, "stash@{0}", "ordinary", day, "2025-10-08"})
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	p := newPrunePrompter(prunePickResult{values: []string{"stash@{0}"}})
	p.beforeDrop = func() {
		if err := recovery.Save(common, recovery.File{Ops: []recovery.Op{{
			ID: "abc", Command: "work.sync", Phase: recovery.PhaseSuspended, StashOID: oid,
		}}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := runPrune(p, "stashes"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "protected and was kept") {
		t.Fatalf("output:\n%s", buf.String())
	}
	for _, c := range gitCalls(m) {
		if strings.HasPrefix(c, "stash drop") {
			t.Fatalf("dropped: %v", gitCalls(m))
		}
	}
}
