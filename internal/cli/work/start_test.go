package work

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/exitcode"
	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var errNoUpstream = fmt.Errorf("no upstream")

func TestResolveStartPoint(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --abbrev-ref main@{upstream}"] = "origin/main"
	m.FailOn["rev-parse --abbrev-ref origin/feat/foo@{upstream}"] = errNoUpstream
	git.Use(m)

	if got := resolveStartPoint("main"); got != "origin/main" {
		t.Fatalf("resolveStartPoint(main) = %q, want origin/main", got)
	}
	if got := resolveStartPoint("origin/feat/foo"); got != "origin/feat/foo" {
		t.Fatalf("resolveStartPoint(remote) = %q, want origin/feat/foo", got)
	}
}

func TestParseStartChangesChoice(t *testing.T) {
	tests := []struct {
		in   string
		mode string
		ok   bool
	}{
		{"", "stash", true},
		{"a", "stash", true},
		{"add", "stash", true},
		{"R", "reset", true},
		{"reset", "reset", true},
		{"c", "cancel", true},
		{"cancel", "cancel", true},
		{"x", "", false},
	}
	for _, tc := range tests {
		mode, ok := parseStartChangesChoice(tc.in)
		if mode != tc.mode || ok != tc.ok {
			t.Errorf("parseStartChangesChoice(%q) = %q, %v; want %q, %v", tc.in, mode, ok, tc.mode, tc.ok)
		}
	}
}

func TestStartRejectsExistingNameBeforeSideEffects(t *testing.T) {
	tests := []struct {
		branches string
		name     string
		want     string
	}{
		{"main", "main", "branch 'main' already exists"},
		{"main", " main ", "branch 'main' already exists"},
		{"feature", "feature/x", "branch 'feature/x' conflicts with existing branch 'feature'"},
		{"a/b", "a", "branch 'a' conflicts with existing branch 'a/b'"},
		{"main", "bad name", "branch name cannot contain ' '"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := setupStart(t, tc.branches)
			m.Repo.Remotes = []string{"origin"}
			m.FailOn["diff-index --quiet HEAD"] = fmt.Errorf("dirty")
			buf := captureText(t)
			err := runStart(prompt.NewNonInteractive(), tc.name)
			assertWorkflow(t, err)
			if buf.String() != tc.want+"\n" {
				t.Fatalf("out=%q", buf.String())
			}
			assertNoStartMutations(t, m)
		})
	}
}

func TestStartCheckRefFormatRejectsBeforeCheckout(t *testing.T) {
	m := setupStart(t, "main")
	buf := captureText(t)
	err := runStart(prompt.NewNonInteractive(), "HEAD")
	assertWorkflow(t, err)
	if buf.String() != "branch name cannot be 'HEAD'\n" {
		t.Fatalf("out=%q", buf.String())
	}
	assertNoStartMutations(t, m)
}

func TestAskReopensPrefilledWhenSubmitRejects(t *testing.T) {
	m := setupStart(t, "main")
	m.FailOn["check-ref-format --branch weird"] = fmt.Errorf("fatal: 'weird' is not a valid branch name")
	buf := captureText(t)
	p := &startPrompter{answers: []string{"weird", "feature"}}
	got, err := askNewBranchName(p, "Branch name", []string{"main"}, "", prepareCaseFold())
	if err != nil {
		t.Fatal(err)
	}
	if got != "feature" {
		t.Fatalf("got %q", got)
	}
	if len(p.defaults) < 2 || p.defaults[0] != "" || p.defaults[1] != "weird" {
		t.Fatalf("defaults=%v", p.defaults)
	}
	if !strings.Contains(buf.String(), "fatal: 'weird' is not a valid branch name") {
		t.Fatalf("out=%q", buf.String())
	}
}

func TestLiveIgnoreCaseResolvesOnDemand(t *testing.T) {
	branches := []string{"main", "feature"}
	for i := 0; i < 40; i++ {
		branches = append(branches, fmt.Sprintf("other%d", i))
	}
	m := setupStart(t, strings.Join(branches, "\n"))
	m.Repo.LocalConfig["core.ignorecase"] = "true"
	m.Outputs["show-ref --verify --quiet refs/heads/Main"] = ""
	fold := prepareCaseFold()
	for _, c := range m.Calls {
		if len(c.Args) > 0 && c.Args[0] == "show-ref" {
			t.Fatalf("prepare spawned %v", c.Args)
		}
	}
	before := len(m.Calls)
	if got := liveBranchProblem("Main", branches, fold); got != "branch 'Main' already exists" {
		t.Fatalf("loose=%q", got)
	}
	if len(m.Calls) != before+1 {
		t.Fatalf("calls=%v", m.Calls[before:])
	}
	before = len(m.Calls)
	if got := liveBranchProblem("MAIN", branches, fold); got != "branch 'MAIN' already exists" {
		t.Fatalf("memo=%q", got)
	}
	if len(m.Calls) != before {
		t.Fatalf("memo spawned %v", m.Calls[before:])
	}
	before = len(m.Calls)
	if got := liveBranchProblem("Feature/x", branches, fold); got != "branch 'Feature/x' conflicts with existing branch 'feature'" {
		t.Fatalf("slash=%q", got)
	}
	if len(m.Calls) != before {
		t.Fatalf("slash spawned %v", m.Calls[before:])
	}
	if got := liveBranchProblem("Feature", branches, fold); got != "" {
		t.Fatalf("packed=%q", got)
	}
	if got := liveBranchProblem("FEATURE", branches, fold); got != "" {
		t.Fatalf("packed memo=%q", got)
	}
	showRefs := 0
	for _, c := range m.Calls[before:] {
		if len(c.Args) > 0 && c.Args[0] == "show-ref" {
			showRefs++
		}
	}
	if showRefs != 1 {
		t.Fatalf("show-ref=%d calls=%v", showRefs, m.Calls[before:])
	}
	before = len(m.Calls)
	if got := liveBranchProblem("-x", branches, fold); got == "" {
		t.Fatal("expected -x rejected")
	}
	if len(m.Calls) != before {
		t.Fatalf("-x spawned %v", m.Calls[before:])
	}
}

func TestPromptGitCallsDoNotGrowWithBranches(t *testing.T) {
	startOpened := func(n int) int {
		t.Helper()
		names := make([]string, n)
		for i := range names {
			names[i] = fmt.Sprintf("branch%d", i)
		}
		m := setupStart(t, strings.Join(names, "\n"))
		m.Repo.LocalConfig["core.ignorecase"] = "true"
		fold := prepareCaseFold()
		var opened int
		seen := false
		p := &startPrompter{answers: []string{"zz-free"}, onString: func() {
			if !seen {
				seen = true
				opened = len(m.Calls)
				for _, c := range m.Calls[:opened] {
					if len(c.Args) > 0 && c.Args[0] == "show-ref" {
						t.Fatalf("show-ref before prompt: %v", c.Args)
					}
				}
			}
		}}
		if _, err := askNewBranchName(p, "Branch name", names, "", fold); err != nil {
			t.Fatal(err)
		}
		if !seen {
			t.Fatal("prompt did not open")
		}
		return opened
	}
	if a, b := startOpened(3), startOpened(120); a != b {
		t.Fatalf("start git calls before prompt: %d vs %d", a, b)
	}

	trackOpened := func(n int) (int, []git.Call) {
		t.Helper()
		names := make([]string, n)
		for i := range names {
			names[i] = fmt.Sprintf("branch%d", i)
		}
		m := trackRunner(t, "origin/x", strings.Join(names, "\n"))
		m.Repo.LocalConfig["core.ignorecase"] = "true"
		var opened int
		seen := false
		p := &trackRecordingPrompter{stringValues: []string{"zz-free"}, onString: func() {
			if !seen {
				seen = true
				opened = len(m.Calls)
			}
		}}
		if err := trackLogic(context.Background(), p, "origin/x", ""); err != nil {
			t.Fatal(err)
		}
		if !seen {
			t.Fatal("prompt did not open")
		}
		return opened, append([]git.Call(nil), m.Calls[:opened]...)
	}
	a, aCalls := trackOpened(3)
	b, bCalls := trackOpened(120)
	if a != b {
		t.Fatalf("track git calls before prompt: %d vs %d", a, b)
	}
	for _, calls := range [][]git.Call{aCalls, bCalls} {
		upstream := 0
		for _, c := range calls {
			joined := strings.Join(c.Args, " ")
			if (len(c.Args) > 0 && c.Args[0] == "show-ref") || strings.Contains(joined, "@{upstream}") {
				t.Fatalf("per-branch git before prompt: %v", c.Args)
			}
			if len(c.Args) >= 2 && c.Args[0] == "for-each-ref" && strings.Contains(c.Args[1], "%(upstream)") {
				upstream++
			}
		}
		if upstream != 1 {
			t.Fatalf("upstream for-each-ref=%d calls=%v", upstream, calls)
		}
	}
}

func TestStartBranchListError(t *testing.T) {
	m := setupStart(t, "main")
	m.FailOn["for-each-ref --format=%(refname:lstrip=2) refs/heads"] = fmt.Errorf("not a git repository")
	err := runStart(prompt.NewNonInteractive(), "topic")
	if err == nil || err.Error() != "not a git repository" {
		t.Fatalf("err=%v", err)
	}
	assertNoStartMutations(t, m)
}

func TestStartCaseOnlyNameProceeds(t *testing.T) {
	m := setupStart(t, "main")
	if err := runStart(prompt.NewNonInteractive(), "Main"); err != nil {
		t.Fatal(err)
	}
	got := checkoutCall(m)
	want := []string{"checkout", "-b", "Main", "--no-track", "main"}
	if len(got) != len(want) {
		t.Fatalf("checkout=%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("checkout=%v", got)
		}
	}
}

func TestStartInteractiveReasksThenStarts(t *testing.T) {
	m := setupStart(t, "main")
	buf := captureText(t)
	p := &startPrompter{answers: []string{"main", "bad name", "", "feature"}}
	if err := runStart(p, "main"); err != nil {
		t.Fatal(err)
	}
	if len(p.defaults) != 4 || p.defaults[0] != "" || p.questions[0] != "Branch name" {
		t.Fatalf("prompts q=%v def=%v", p.questions, p.defaults)
	}
	out := buf.String()
	if count := bytes.Count([]byte(out), []byte("branch 'main' already exists")); count != 2 {
		t.Fatalf("exists count=%d out=%q", count, out)
	}
	for _, want := range []string{"branch name cannot contain ' '", "branch name is required"} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	got := checkoutCall(m)
	want := []string{"checkout", "-b", "feature", "--no-track", "main"}
	if len(got) != len(want) {
		t.Fatalf("checkout=%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("checkout=%v", got)
		}
	}
}

func TestStartPromptedDuplicateReasks(t *testing.T) {
	m := setupStart(t, "main")
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "main\t"
	buf := captureText(t)
	p := &startPrompter{answers: []string{"main", "feature"}, pickAns: "main"}
	p.onPick = func() {
		if len(p.questions) != 2 {
			t.Fatalf("from-ref asked before the name was accepted: %v", p.questions)
		}
	}
	if err := runStart(p); err != nil {
		t.Fatal(err)
	}
	if strings.Join(p.events, ",") != "string,string,pick" {
		t.Fatalf("order=%v", p.events)
	}
	if !bytes.Contains(buf.Bytes(), []byte("branch 'main' already exists")) {
		t.Fatalf("out=%q", buf.String())
	}
	sawFetch := false
	for _, c := range m.Calls {
		if len(c.Args) > 0 && c.Args[0] == "fetch" {
			sawFetch = true
		}
	}
	if !sawFetch {
		t.Fatal("expected fetch while choosing where to start")
	}
	got := checkoutCall(m)
	if len(got) < 3 || got[2] != "feature" {
		t.Fatalf("checkout=%v", got)
	}
}

func TestStartInteractiveCancel(t *testing.T) {
	m := setupStart(t, "main")
	p := &startPrompter{stringErr: prompt.ErrUserCancelled}
	err := runStart(p, "main")
	if !errors.Is(err, prompt.ErrUserCancelled) {
		t.Fatalf("err=%v", err)
	}
	assertNoStartMutations(t, m)
}

func TestStartIgnoreCaseLooseNameRejected(t *testing.T) {
	m := setupStart(t, "main")
	m.Repo.LocalConfig["core.ignorecase"] = "true"
	m.Outputs["show-ref --verify --quiet refs/heads/Main"] = ""
	buf := captureText(t)
	err := runStart(prompt.NewNonInteractive(), "Main")
	assertWorkflow(t, err)
	if buf.String() != "branch 'Main' already exists\n" {
		t.Fatalf("out=%q", buf.String())
	}
	assertNoStartMutations(t, m)
}

func TestStartIgnoreCasePackedNameProceeds(t *testing.T) {
	m := setupStart(t, "main")
	m.Repo.LocalConfig["core.ignorecase"] = "true"
	if err := runStart(prompt.NewNonInteractive(), "Main"); err != nil {
		t.Fatal(err)
	}
	got := checkoutCall(m)
	if len(got) < 3 || got[2] != "Main" {
		t.Fatalf("checkout=%v", got)
	}
}

func TestStartIgnoreCaseSlashConflict(t *testing.T) {
	m := setupStart(t, "feature\na/b")
	m.Repo.LocalConfig["core.ignorecase"] = "true"
	buf := captureText(t)
	err := runStart(prompt.NewNonInteractive(), "Feature/x")
	assertWorkflow(t, err)
	if buf.String() != "branch 'Feature/x' conflicts with existing branch 'feature'\n" {
		t.Fatalf("out=%q", buf.String())
	}
	assertNoStartMutations(t, m)

	buf.Reset()
	err = runStart(prompt.NewNonInteractive(), "A")
	assertWorkflow(t, err)
	if buf.String() != "branch 'A' conflicts with existing branch 'a/b'\n" {
		t.Fatalf("out=%q", buf.String())
	}
	assertNoStartMutations(t, m)
}

func TestStartEndOfInput(t *testing.T) {
	m := setupStart(t, "main")
	p := &startPrompter{stringErr: io.EOF}
	err := runStart(p, "main")
	if err == nil || err.Error() != "end of input" {
		t.Fatalf("err=%v", err)
	}
	assertNoStartMutations(t, m)
}

func TestStartLooseCaseOnlyRealRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	dir := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(dir, "none"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(dir, "state.json"))
	if err := os.WriteFile(filepath.Join(dir, "none"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	gitCmd("init", "-b", "main")
	gitCmd("config", "user.email", "test@example.com")
	gitCmd("config", "user.name", "Test")
	gitCmd("config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("o\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd("add", "f")
	gitCmd("commit", "-q", "-m", "o")
	ignore, err := exec.Command("git", "-C", dir, "config", "--get", "core.ignorecase").Output()
	if err != nil {
		t.Fatal(err)
	}
	git.Use(git.RealRunner{})
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	t.Chdir(dir)
	buf := captureText(t)
	err = runStart(prompt.NewNonInteractive(), "Main")
	if strings.TrimSpace(string(ignore)) == "true" {
		assertWorkflow(t, err)
		if !strings.Contains(buf.String(), "branch 'Main' already exists") {
			t.Fatalf("out=%q", buf.String())
		}
		stash, sErr := exec.Command("git", "-C", dir, "stash", "list").Output()
		if sErr != nil {
			t.Fatal(sErr)
		}
		if strings.TrimSpace(string(stash)) != "" {
			t.Fatalf("stash=%q", stash)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
}

func setupStart(t *testing.T, branches string) *git.MemoryRunner {
	t.Helper()
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(gitDir, "elegant-git", "state.json"))
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	m.Outputs["for-each-ref --format=%(refname:lstrip=2) refs/heads"] = branches
	if err := memrepo.Save(gitDir, &memrepo.State{DefaultBranch: "main"}); err != nil {
		t.Fatal(err)
	}
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	return m
}

func runStart(p prompt.Prompter, args ...string) error {
	var name, fromRef string
	spec := startSpec(&name, &fromRef)
	cmd := &cobra.Command{}
	cmd.SetContext(prompt.WithPrompter(context.Background(), p))
	return startRun(cmd, args, spec)
}

func captureText(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	text.SetOutput(&buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	return &buf
}

func assertWorkflow(t *testing.T, err error) {
	t.Helper()
	var es cliruntime.ExitStatus
	if !errors.As(err, &es) || es.Code != exitcode.WorkflowError {
		t.Fatalf("err=%v", err)
	}
}

func assertNoStartMutations(t *testing.T, m *git.MemoryRunner) {
	t.Helper()
	for _, c := range m.Calls {
		if len(c.Args) == 0 {
			continue
		}
		switch c.Args[0] {
		case "fetch", "stash", "reset", "checkout", "update-index", "diff-index":
			t.Fatalf("unexpected git %v", c.Args)
		}
	}
}

func checkoutCall(m *git.MemoryRunner) []string {
	for _, c := range m.Calls {
		if len(c.Args) > 0 && c.Args[0] == "checkout" {
			return c.Args
		}
	}
	return nil
}

type startPrompter struct {
	answers   []string
	pickAns   string
	stringErr error
	errOn     int
	n         int
	questions []string
	defaults  []string
	picks     int
	events    []string
	onPick    func()
	onString  func()
}

func (p *startPrompter) String(question, defaultVal string) (string, error) {
	if p.onString != nil {
		p.onString()
	}
	p.events = append(p.events, "string")
	p.questions = append(p.questions, question)
	p.defaults = append(p.defaults, defaultVal)
	if p.stringErr != nil && p.n == p.errOn {
		p.n++
		return "", p.stringErr
	}
	if p.n >= len(p.answers) {
		return "", io.EOF
	}
	ans := p.answers[p.n]
	p.n++
	return ans, nil
}

func (p *startPrompter) Confirm(string, bool) (bool, error) {
	return false, fmt.Errorf("unexpected confirm")
}
func (p *startPrompter) Choose(string, []string) (int, error) {
	return -1, fmt.Errorf("unexpected choose")
}
func (p *startPrompter) Required(string, string) error { return fmt.Errorf("unexpected required") }
func (p *startPrompter) EditOrAccept(string, string) (string, error) {
	return "", fmt.Errorf("unexpected edit")
}
func (p *startPrompter) Optional(string, string) (string, error) {
	return "", fmt.Errorf("unexpected optional")
}
func (p *startPrompter) Closed(string, []string, string, bool) (string, error) {
	return "", fmt.Errorf("unexpected closed")
}
func (p *startPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, fmt.Errorf("unexpected batch")
}

func (p *startPrompter) Pick(string, []prompt.Choice, string) (string, error) {
	if p.onPick != nil {
		p.onPick()
	}
	p.events = append(p.events, "pick")
	p.picks++
	if p.pickAns == "" {
		return "", fmt.Errorf("unexpected pick")
	}
	return p.pickAns, nil
}
