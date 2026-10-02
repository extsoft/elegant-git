package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/cli/catalog"
	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

func TestRootHelpListsObjects(t *testing.T) {
	bin := buildTestBinary(t)
	out, err := exec.Command(bin, "--help").Output()
	if err != nil {
		t.Fatalf("--help: %v", err)
	}
	text := string(out)
	if !strings.Contains(text, "Objects:") {
		t.Error("help missing Objects section")
	}
	for _, heading := range []string{"self —", "repo —", "work —", "hook —", "release —", "workspace —"} {
		if !strings.Contains(text, heading) {
			t.Errorf("help missing heading %q", heading)
		}
	}
	if strings.Contains(text, "  git —") || strings.Contains(text, "  memory —") {
		t.Error("help still lists git or memory objects")
	}
	for _, tc := range []struct{ object, action string }{
		{"self", "configure"},
		{"repo", "clone"},
		{"work", "start"},
		{"hook", "list"},
		{"release", "notes"},
	} {
		found := false
		for _, row := range sectionRows(text, "  "+tc.object+" —", "    ") {
			if firstField(strings.TrimSpace(row)) == tc.action {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("help missing action %q under %s", tc.action, tc.object)
		}
	}
}

func TestBareEgOutsideRepoPrintsHelp(t *testing.T) {
	bin := buildTestBinary(t)
	dir := t.TempDir()
	cmd := exec.Command(bin)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CEILING_DIRECTORIES="+dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bare eg: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Objects:") {
		t.Fatalf("expected help, got: %s", out)
	}
}

func TestBareEgInRepoNonInteractivePrintsHelp(t *testing.T) {
	bin := buildTestBinary(t)
	dir := t.TempDir()
	init := exec.Command("git", "init")
	init.Dir = dir
	if out, err := init.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	cmd := exec.Command(bin, "--non-interactive")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bare eg: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Objects:") {
		t.Fatalf("expected help, got: %s", out)
	}
}

func TestBareEgPickerRunsPickedCommand(t *testing.T) {
	_, dir := pickerRepo(t)
	p := &rootPrompter{picks: []string{"work list"}}
	out := runRootWithPrompter(t, p)
	if !slices.Equal(choiceValues(p.choices[0]), pickerChoices()) {
		t.Fatalf("choices %v", choiceValues(p.choices[0]))
	}
	if p.labels[0] != filepath.Base(dir) {
		t.Fatalf("label %q", p.labels[0])
	}
	if p.defs[0] != "work sync" {
		t.Fatalf("default %q", p.defs[0])
	}
	sync := p.choices[0][0]
	if sync.Value != "work sync" || sync.Description != catalog.Purpose("work", "sync") {
		t.Fatalf("first choice %+v", sync)
	}
	if !strings.Contains(out, "Branch") {
		t.Fatalf("work list did not run: %q", out)
	}
}

func TestBareEgGitFailureReturnsError(t *testing.T) {
	m := git.NewMemoryRunner()
	m.FailOn["rev-parse --is-inside-work-tree"] = fmt.Errorf("detected dubious ownership")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetContext(context.Background())
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetContext(context.Background())
	})
	err := runRoot(rootCmd, nil)
	if err == nil {
		t.Fatal("expected git error")
	}
	if strings.Contains(buf.String(), "Objects:") {
		t.Fatalf("printed help: %s", buf.String())
	}
}

func TestBareEgNotRepoMessagePrintsHelp(t *testing.T) {
	m := git.NewMemoryRunner()
	m.FailOn["rev-parse --is-inside-work-tree"] = fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetContext(context.Background())
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetContext(context.Background())
	})
	if err := runRoot(rootCmd, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Objects:") {
		t.Fatalf("expected help, got: %s", buf.String())
	}
}

func TestBareEgNonInteractiveSkipsGitFailure(t *testing.T) {
	m := git.NewMemoryRunner()
	m.FailOn["rev-parse --is-inside-work-tree"] = fmt.Errorf("detected dubious ownership")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetContext(context.Background())
	})
	if err := runRoot(rootCmd, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Objects:") {
		t.Fatalf("expected help, got: %s", buf.String())
	}
	if len(m.Calls) != 0 {
		t.Fatalf("git called: %+v", m.Calls)
	}
}

func TestBareEgNotWorkTreePrintsHelp(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --is-inside-work-tree"] = "false"
	m.FailOn["diff-index --quiet HEAD"] = fmt.Errorf("dirty")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetContext(context.Background())
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetContext(context.Background())
	})
	if err := runRoot(rootCmd, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Objects:") {
		t.Fatalf("expected help, got: %s", buf.String())
	}
	if hasGitCall(m, "add", "--interactive") {
		t.Fatal("ran save outside a work tree")
	}
}

type cLocaleRunner struct {
	*git.MemoryRunner
	locale  string
	userErr error
}

func (r *cLocaleRunner) OutputLocale(locale string, args ...string) (string, error) {
	if strings.Join(args, " ") == "rev-parse --is-inside-work-tree" {
		r.locale = locale
		if r.userErr != nil {
			return "", fmt.Errorf("detected dubious ownership")
		}
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}
	return r.MemoryRunner.Output(args...)
}

func (r *cLocaleRunner) Output(args ...string) (string, error) {
	if r.userErr != nil && strings.Join(args, " ") == "rev-parse --is-inside-work-tree" {
		return "", r.userErr
	}
	return r.MemoryRunner.Output(args...)
}

func TestBareEgRepoCheckUsesCLocale(t *testing.T) {
	t.Setenv("LC_ALL", "de_DE.UTF-8")
	m := &cLocaleRunner{MemoryRunner: git.NewMemoryRunner()}
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetContext(context.Background())
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetContext(context.Background())
	})
	if err := runRoot(rootCmd, nil); err != nil {
		t.Fatal(err)
	}
	if m.locale != "C" {
		t.Fatalf("locale during check %q", m.locale)
	}
	if os.Getenv("LC_ALL") != "de_DE.UTF-8" {
		t.Fatalf("LC_ALL left as %q", os.Getenv("LC_ALL"))
	}
	if !strings.Contains(buf.String(), "Objects:") {
		t.Fatalf("expected help, got: %s", buf.String())
	}
}

func TestBareEgGitFailureUsesUserLocale(t *testing.T) {
	t.Setenv("LC_ALL", "de_DE.UTF-8")
	m := &cLocaleRunner{
		MemoryRunner: git.NewMemoryRunner(),
		userErr:      fmt.Errorf("fatal: zweifelhafte Eigentumsverhältnisse"),
	}
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetContext(context.Background())
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetContext(context.Background())
	})
	err := runRoot(rootCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "zweifelhafte") {
		t.Fatalf("err %v", err)
	}
	if m.locale != "C" || os.Getenv("LC_ALL") != "de_DE.UTF-8" {
		t.Fatalf("locale %q env %q", m.locale, os.Getenv("LC_ALL"))
	}
	if strings.Contains(buf.String(), "Objects:") {
		t.Fatalf("printed help: %s", buf.String())
	}
}

func TestBareEgUnbornHeadPrintsHelp(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --is-inside-work-tree"] = "true"
	m.FailOn["rev-parse --verify HEAD^{commit}"] = fmt.Errorf("fatal: Needed a single revision")
	m.FailOn["diff-index --quiet HEAD"] = fmt.Errorf("dirty")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetContext(context.Background())
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetContext(context.Background())
	})
	if err := runRoot(rootCmd, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Objects:") {
		t.Fatalf("expected help, got: %s", buf.String())
	}
	if hasGitCall(m, "add", "--interactive") {
		t.Fatal("ran save with no commits")
	}
}

func TestBareEgBrokenHeadReturnsError(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --is-inside-work-tree"] = "true"
	m.Outputs["rev-list --all -1"] = "abc"
	m.FailOn["rev-parse --verify HEAD^{commit}"] = fmt.Errorf("fatal: Needed a single revision")
	m.FailOn["rev-list -1 HEAD"] = fmt.Errorf("fatal: bad object HEAD")
	m.FailOn["diff-index --quiet HEAD"] = fmt.Errorf("dirty")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetContext(context.Background())
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetContext(context.Background())
	})
	err := runRoot(rootCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "bad object HEAD") {
		t.Fatalf("err %v", err)
	}
	if strings.Contains(buf.String(), "Objects:") {
		t.Fatalf("printed help: %s", buf.String())
	}
	if hasGitCall(m, "add", "--interactive") {
		t.Fatal("ran save with a broken HEAD")
	}
}

func TestBareEgPickerLabelFallsBack(t *testing.T) {
	m, _ := pickerRepo(t)
	m.FailOn["rev-parse --show-toplevel"] = fmt.Errorf("fatal: this operation must be run in a work tree")
	p := &rootPrompter{picks: []string{"quit"}}
	runRootWithPrompter(t, p)
	if len(p.labels) != 1 || p.labels[0] != "eg" {
		t.Fatalf("label %v", p.labels)
	}
}

func TestBareEgListThenPickerSkipsList(t *testing.T) {
	m, _ := pickerRepo(t)
	m.Outputs["rev-list --left-right --count feat...main"] = "0\t0"
	p := &rootPrompter{picks: []string{"quit"}}
	runRootWithPrompter(t, p)
	if len(p.defs) != 1 || p.defs[0] != "work sync" {
		t.Fatalf("default %v choices %v", p.defs, choiceValues(p.choices[0]))
	}
}

func TestBareEgDirtyFeatureRunsSave(t *testing.T) {
	m, _ := pickerRepo(t)
	m.FailOn["diff-index --quiet HEAD"] = fmt.Errorf("dirty")
	p := &rootPrompter{}
	runRootWithPrompter(t, p)
	if len(p.labels) != 0 {
		t.Fatalf("prompted %v", p.labels)
	}
	if !hasGitCall(m, "add", "--interactive") || !hasGitCall(m, "commit") {
		t.Fatalf("calls %+v", m.Calls)
	}
}

func TestBareEgSyncThenPickerSkipsSync(t *testing.T) {
	m, dir := pickerRepo(t)
	stubSyncBranchSources(t, m, dir, "feat")
	m.Outputs["rev-list --left-right --count HEAD...@{upstream}"] = "0\t2"
	m.Outputs["rev-parse --abbrev-ref feat@{upstream}"] = "origin/feat"
	p := &rootPrompter{picks: []string{"origin/main", "quit"}}
	runRootWithPrompter(t, p)
	if len(p.choices) != 2 {
		t.Fatalf("choices %v", len(p.choices))
	}
	if p.labels[0] != "Branch name" {
		t.Fatalf("labels %v", p.labels)
	}
	if len(p.choices[1]) == 0 || p.choices[1][0].Value != "work sync" {
		t.Fatalf("action choices %v", choiceValues(p.choices[1]))
	}
	if len(p.defs) != 2 || p.defs[1] != "work start" {
		t.Fatalf("defaults %v", p.defs)
	}
}

func TestBareEgProtectedDivergedInsertsSync(t *testing.T) {
	m, _ := pickerRepo(t)
	m.Repo.CurrentBranch = "main"
	m.Outputs["rev-list --left-right --count HEAD...@{upstream}"] = "1\t2"
	p := &rootPrompter{picks: []string{"quit"}}
	runRootWithPrompter(t, p)
	if len(p.choices) != 1 || len(p.choices[0]) == 0 || p.choices[0][0].Value != "work sync" {
		t.Fatalf("choices %v", p.choices)
	}
	if len(p.defs) != 1 || p.defs[0] != "work sync" {
		t.Fatalf("default %v", p.defs)
	}
}

func TestBareEgAcceptPassesBranch(t *testing.T) {
	m, _ := pickerRepo(t)
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "feat"
	p := &rootPrompter{picks: []string{"work accept"}}
	_, err := runRootErr(t, p)
	if !hasGitCall(m, "checkout", "-B", "__eg", "feat") {
		t.Fatalf("accept branch: err=%v calls=%+v", err, m.Calls)
	}
}

func TestBareEgPickerCancelReturnsError(t *testing.T) {
	pickerRepo(t)
	p := &rootPrompter{pickErr: prompt.ErrUserCancelled}
	_, err := runRootErr(t, p)
	if !errors.Is(err, prompt.ErrUserCancelled) {
		t.Fatalf("err %v", err)
	}
}

func TestBareEgPickerQuitRunsNothing(t *testing.T) {
	pickerRepo(t)
	p := &rootPrompter{picks: []string{"quit"}}
	out := runRootWithPrompter(t, p)
	if !slices.Equal(choiceValues(p.choices[0]), pickerChoices()) {
		t.Fatalf("choices %v", choiceValues(p.choices[0]))
	}
	if strings.Contains(out, "Branch") || strings.Contains(out, "Objects:") {
		t.Fatalf("quit ran something: %q", out)
	}
}

func TestBareEgPickerHelpThenQuit(t *testing.T) {
	pickerRepo(t)
	p := &rootPrompter{picks: []string{"help", "quit"}}
	out := runRootWithPrompter(t, p)
	if len(p.labels) != 2 {
		t.Fatalf("asked %v", p.labels)
	}
	if !strings.Contains(out, "Objects:") {
		t.Fatalf("help missing: %q", out)
	}
	if strings.Contains(out, "Branch") {
		t.Fatalf("help ran a command: %q", out)
	}
}

func pickerRepo(t *testing.T) (*git.MemoryRunner, string) {
	t.Helper()
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
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	return m, dir
}

func stubSyncBranchSources(t *testing.T, m *git.MemoryRunner, dir, currentBranch string) {
	t.Helper()
	gitDir := filepath.Join(dir, ".git")
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(gitDir, "elegant-git", "state.json"))
	m.Outputs["rev-parse --git-dir"] = gitDir
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["rev-parse --verify --quiet main"] = "abc"
	m.Outputs["rev-parse --verify --quiet origin/main"] = "abc"
	m.Outputs["rev-parse --verify --quiet --abbrev-ref --branches=refs/heads "+currentBranch] = currentBranch
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = currentBranch + "\torigin/feat\nmain\t"
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/feat\norigin/main"
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{currentBranch: "main"},
	}); err != nil {
		t.Fatal(err)
	}
}

func hasGitCall(m *git.MemoryRunner, args ...string) bool {
	for _, c := range m.Calls {
		if slices.Equal(c.Args, args) {
			return true
		}
	}
	return false
}

func pickerChoices() []string {
	return []string{
		"work sync", "work start", "work accept", "work list",
		"repo list", "repo sync", "repo prune", "repo configure", "repo doctor",
		"workspace new",
		"hook list", "hook new", "hook edit",
		"release new", "release notes",
		"help", "quit",
	}
}

func choiceValues(choices []prompt.Choice) []string {
	out := make([]string, len(choices))
	for i, c := range choices {
		out[i] = c.Value
	}
	return out
}

func runRootWithPrompter(t *testing.T, p *rootPrompter) string {
	t.Helper()
	out, err := runRootErr(t, p)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func runRootErr(t *testing.T, p *rootPrompter) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetContext(prompt.WithPrompter(context.Background(), p))
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetContext(context.Background())
	})
	err := runRoot(rootCmd, nil)
	return buf.String(), err
}

type rootPrompter struct {
	picks   []string
	pickErr error
	idx     int
	labels  []string
	defs    []string
	choices [][]prompt.Choice
}

func (p *rootPrompter) String(string, string) (string, error) { return "", prompt.ErrNonInteractive }
func (p *rootPrompter) Confirm(string, bool) (bool, error)    { return false, nil }
func (p *rootPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}
func (p *rootPrompter) Pick(label string, choices []prompt.Choice, def string) (string, error) {
	p.labels = append(p.labels, label)
	p.defs = append(p.defs, def)
	p.choices = append(p.choices, append([]prompt.Choice(nil), choices...))
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
func (p *rootPrompter) Required(string, string) error { return nil }
func (p *rootPrompter) EditOrAccept(string, string) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *rootPrompter) Optional(string, string) (string, error) { return "", nil }
func (p *rootPrompter) Closed(string, []string, string, bool) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *rootPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}

func TestUnknownCommandExit46(t *testing.T) {
	bin := buildTestBinary(t)
	cmd := exec.Command(bin, "nosuch-command")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error exit")
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 46 {
		t.Fatalf("exit code = %v, want 46", err)
	}
}

func TestUnknownFlagShowsHelp(t *testing.T) {
	bin := buildTestBinary(t)
	cmd := exec.Command(bin, "workspace", "list", "--unknown-flag")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error exit")
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 47 {
		t.Fatalf("exit code = %v, want 47; output: %s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "unknown flag") && !strings.Contains(text, "Unknown flag") {
		t.Fatalf("expected unknown flag error, got: %s", text)
	}
	if !strings.Contains(text, "Usage:") {
		t.Fatalf("expected help after error, got: %s", text)
	}
}

func TestExtraPositionalShowsHelp(t *testing.T) {
	bin := buildTestBinary(t)
	cmd := exec.Command(bin, "version", "extra-arg")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error exit")
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 47 {
		t.Fatalf("exit code = %v, want 47; output: %s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "accepts no arguments") {
		t.Fatalf("expected extra arg error, got: %s", text)
	}
	if !strings.Contains(text, "Usage:") {
		t.Fatalf("expected help after error, got: %s", text)
	}
}

func TestVersionFlag(t *testing.T) {
	bin := buildTestBinary(t)
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		t.Fatal("empty version output")
	}
}

func TestVersionSubcommand(t *testing.T) {
	bin := buildTestBinary(t)
	out, err := exec.Command(bin, "version").Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		t.Fatal("empty version output")
	}
}

func TestNoWorkflowsFlagAccepted(t *testing.T) {
	bin := buildTestBinary(t)
	out, err := exec.Command(bin, "--no-workflows", "--help").Output()
	if err != nil {
		t.Fatalf("--no-workflows --help: %v", err)
	}
	if !strings.Contains(string(out), "--no-workflows") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestObjectGroupHelpHook(t *testing.T) {
	bin := buildTestBinary(t)
	out, err := exec.Command(bin, "hook").Output()
	if err != nil {
		t.Fatalf("hook: %v", err)
	}
	text := string(out)
	if strings.Contains(text, "Objects:") && strings.Contains(text, "  self —") {
		t.Fatal("hook without subcommand should not show full root catalog")
	}
	for _, want := range []string{"hook — manage command hooks", "list", "new", "edit", ".config/elegant-git/hooks"} {
		if !strings.Contains(text, want) {
			t.Errorf("hook help missing %q", want)
		}
	}
}

func TestSkipAuto(t *testing.T) {
	version := &cobra.Command{Use: "version"}
	if !skipAuto(version) {
		t.Fatal("version")
	}
	completion := &cobra.Command{Use: "completion"}
	bash := &cobra.Command{Use: "bash"}
	completion.AddCommand(bash)
	if !skipAuto(bash) {
		t.Fatal("completion child")
	}
	help := &cobra.Command{Use: "help"}
	if !skipAuto(help) {
		t.Fatal("help")
	}
	migrate := &cobra.Command{Use: "migrate"}
	migrate.Flags().Bool("dry-run", false, "")
	if err := migrate.Flags().Set("dry-run", "true"); err != nil {
		t.Fatal(err)
	}
	if !skipAuto(migrate) {
		t.Fatal("migrate --dry-run")
	}
	if err := migrate.Flags().Set("dry-run", "false"); err != nil {
		t.Fatal(err)
	}
	if skipAuto(migrate) {
		t.Fatal("migrate without dry-run must run auto")
	}
	start := &cobra.Command{Use: "start"}
	if skipAuto(start) {
		t.Fatal("normal command")
	}
}

func TestShouldAutoConfigure(t *testing.T) {
	start := &cobra.Command{Use: "start"}
	if !shouldAutoConfigure(start, false) {
		t.Fatal("normal command")
	}
	if shouldAutoConfigure(start, true) {
		t.Fatal("nested")
	}
	version := &cobra.Command{Use: "version"}
	if shouldAutoConfigure(version, false) {
		t.Fatal("version")
	}
}

func TestInvocationNested(t *testing.T) {
	t.Setenv("ELEGANT_GIT_DEPTH", "")
	if invocationNested() {
		t.Fatal("unset")
	}
	t.Setenv("ELEGANT_GIT_DEPTH", "0")
	if invocationNested() {
		t.Fatal("zero")
	}
	t.Setenv("ELEGANT_GIT_DEPTH", "1")
	if !invocationNested() {
		t.Fatal("parent depth")
	}
}

func TestLegacyShimStartWork(t *testing.T) {
	bin := buildTestBinary(t)
	out, err := exec.Command(bin, "start-work", "--help").Output()
	if err != nil {
		t.Fatalf("start-work --help: %v", err)
	}
	if !strings.Contains(string(out), "work start") && !strings.Contains(string(out), "start") {
		t.Fatalf("expected work start help, got: %s", out)
	}
}

func buildTestBinary(t *testing.T) string {
	t.Helper()
	stateDir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(stateDir, "state.json"))
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(stateDir, "repo-state.json"))
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(stateDir, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("ELEGANT_GIT_DEPTH", "")
	dir := t.TempDir()
	bin := filepath.Join(dir, "eg")
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/eg")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}
