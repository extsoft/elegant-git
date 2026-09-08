package work

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

func TestSaveCommandHelpExplainsTarget(t *testing.T) {
	var buf strings.Builder
	cliruntime.PrintCommandHelp(&buf, newSaveCommand())
	out := buf.String()
	for _, want := range []string{
		"save [target]",
		"unique commit",
		"new",
		"HEAD",
		"amended",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q:\n%s", want, out)
		}
	}
}

func TestSaveTargetChoicesUsesSourceBranch(t *testing.T) {
	m := setupSave(t)
	now := int64(1_700_000_000)
	saveNowUnix = func() int64 { return now }
	t.Cleanup(func() { saveNowUnix = func() int64 { return time.Now().Unix() } })
	m.Outputs["log --format="+saveCommitLogFormat+" main..feature"] = fmt.Sprintf(
		"aaa\t%d\tnewest subject\nbbb\t%d\tolder subject",
		now-2*3600, now-3*86400,
	)
	got := saveTargetChoices("feature")
	want := []prompt.Choice{
		{Value: "new", Description: "Create a new commit."},
		{Value: "aaa", Display: "[ 2 h ago]  aaa", Description: "newest subject"},
		{Value: "bbb", Display: "[ 3 d ago]  bbb", Description: "older subject"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("choices %#v, want %#v", got, want)
	}
	if saveRefFromChoice("[ 2 h ago]  aaa") != "aaa" {
		t.Fatal("saveRefFromChoice should keep the hash")
	}
}

func TestFormatSaveRelTimePadsNumber(t *testing.T) {
	now := int64(1_700_000_000)
	tests := []struct {
		delta int64
		want  string
	}{
		{5, "[ 5 s ago]"},
		{12, "[12 s ago]"},
		{2 * 60, "[ 2 m ago]"},
		{45 * 60, "[45 m ago]"},
		{2 * 3600, "[ 2 h ago]"},
		{12 * 3600, "[12 h ago]"},
		{3 * 86400, "[ 3 d ago]"},
		{12 * 86400, "[12 d ago]"},
	}
	for _, tc := range tests {
		got := formatSaveRelTime(now-tc.delta, now)
		if got != tc.want {
			t.Errorf("delta %d: got %q want %q", tc.delta, got, tc.want)
		}
	}
}

func TestSaveCommitNoUniqueCommits(t *testing.T) {
	m := setupSave(t)
	p := &sessionPrompter{picks: []string{"should-not-be-used"}}
	if err := runSaveCommit(p, nil); err != nil {
		t.Fatal(err)
	}
	if len(p.asked) != 0 {
		t.Fatalf("unexpected pick: %v", p.asked)
	}
	got := saveGitActions(m)
	want := []string{"add --interactive", "commit"}
	if !slices.Equal(got, want) {
		t.Fatalf("git %v, want %v", got, want)
	}
}

func TestSaveCommitLogFailureSkipsPicker(t *testing.T) {
	m := setupSave(t)
	m.FailOn["log --format="+saveCommitLogFormat+" main..feature"] = errors.New("fatal: bad revision")
	p := &sessionPrompter{picks: []string{"should-not-be-used"}}
	if err := runSaveCommit(p, nil); err != nil {
		t.Fatal(err)
	}
	if len(p.asked) != 0 {
		t.Fatalf("unexpected pick: %v", p.asked)
	}
	got := saveGitActions(m)
	want := []string{"add --interactive", "commit"}
	if !slices.Equal(got, want) {
		t.Fatalf("git %v, want %v", got, want)
	}
}

func TestSaveCommitPickerOrderNewThenNewestFirst(t *testing.T) {
	m := setupSave(t)
	now := int64(1_700_000_000)
	saveNowUnix = func() int64 { return now }
	t.Cleanup(func() { saveNowUnix = func() int64 { return time.Now().Unix() } })
	stubUniqueCommits(m, "aaa", "bbb")
	p := &sessionPrompter{picks: []string{"new"}}
	if err := runSaveCommit(p, nil); err != nil {
		t.Fatal(err)
	}
	if len(p.pickChoices) != 1 {
		t.Fatalf("picks=%d", len(p.pickChoices))
	}
	got := choiceValues(p.pickChoices[0])
	want := []string{"new", "aaa", "bbb"}
	if !slices.Equal(got, want) {
		t.Fatalf("choices %v, want %v", got, want)
	}
	gotDisplay := choiceDisplays(p.pickChoices[0])
	wantDisplay := []string{"", "[ 2 h ago]  aaa", "[ 3 d ago]  bbb"}
	if !slices.Equal(gotDisplay, wantDisplay) {
		t.Fatalf("display %v, want %v", gotDisplay, wantDisplay)
	}
}

func TestSaveCommitPickNew(t *testing.T) {
	m := setupSave(t)
	stubUniqueCommits(m, "aaa", "bbb")
	p := &sessionPrompter{picks: []string{"new"}}
	if err := runSaveCommit(p, nil); err != nil {
		t.Fatal(err)
	}
	got := saveGitActions(m)
	want := []string{"add --interactive", "commit"}
	if !slices.Equal(got, want) {
		t.Fatalf("git %v, want %v", got, want)
	}
}

func TestSaveCommitPickHEADAmends(t *testing.T) {
	m := setupSave(t)
	stubUniqueCommits(m, "aaa", "bbb")
	p := &sessionPrompter{picks: []string{"[ 2 h ago]  aaa"}}
	if err := runSaveCommit(p, nil); err != nil {
		t.Fatal(err)
	}
	got := saveGitActions(m)
	want := []string{"add --interactive", "commit --amend"}
	if !slices.Equal(got, want) {
		t.Fatalf("git %v, want %v", got, want)
	}
}

func TestSaveCommitPickOlderFixupAutosquash(t *testing.T) {
	m := setupSave(t)
	stubUniqueCommits(m, "aaa", "bbb")
	p := &sessionPrompter{picks: []string{"bbb"}}
	if err := runSaveCommit(p, nil); err != nil {
		t.Fatal(err)
	}
	got := saveGitActions(m)
	want := []string{
		"add --interactive",
		"commit --fixup=bbb",
		"-c sequence.editor=true rebase --interactive --autosquash --autostash parent",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("git %v, want %v", got, want)
	}
}

func TestSaveCommitNonInteractiveSkipsPicker(t *testing.T) {
	m := setupSave(t)
	stubUniqueCommits(m, "aaa", "bbb")
	if err := runSaveCommit(prompt.NewNonInteractive(), nil); err != nil {
		t.Fatal(err)
	}
	got := saveGitActions(m)
	want := []string{"add --interactive", "commit"}
	if !slices.Equal(got, want) {
		t.Fatalf("git %v, want %v", got, want)
	}
}

func TestSaveCommitNonInteractiveTargetNew(t *testing.T) {
	m := setupSave(t)
	stubUniqueCommits(m, "aaa", "bbb")
	if err := runSaveCommit(prompt.NewNonInteractive(), []string{"new"}); err != nil {
		t.Fatal(err)
	}
	got := saveGitActions(m)
	want := []string{"add --interactive", "commit"}
	if !slices.Equal(got, want) {
		t.Fatalf("git %v, want %v", got, want)
	}
}

func TestSaveCommitNonInteractiveTargetHEAD(t *testing.T) {
	m := setupSave(t)
	stubUniqueCommits(m, "aaa", "bbb")
	if err := runSaveCommit(prompt.NewNonInteractive(), []string{"aaa"}); err != nil {
		t.Fatal(err)
	}
	got := saveGitActions(m)
	want := []string{"add --interactive", "commit --amend"}
	if !slices.Equal(got, want) {
		t.Fatalf("git %v, want %v", got, want)
	}
}

func TestSaveCommitNonInteractiveTargetHEADWord(t *testing.T) {
	m := setupSave(t)
	stubUniqueCommits(m, "aaa", "bbb")
	if err := runSaveCommit(prompt.NewNonInteractive(), []string{"HEAD"}); err != nil {
		t.Fatal(err)
	}
	got := saveGitActions(m)
	want := []string{"add --interactive", "commit --amend"}
	if !slices.Equal(got, want) {
		t.Fatalf("git %v, want %v", got, want)
	}
}

func TestSaveCommitNonInteractiveTargetFullSHA(t *testing.T) {
	m := setupSave(t)
	stubUniqueCommits(m, "aaa", "bbb")
	full := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	m.Outputs["rev-parse "+full] = "aaa"
	if err := runSaveCommit(prompt.NewNonInteractive(), []string{full}); err != nil {
		t.Fatal(err)
	}
	got := saveGitActions(m)
	want := []string{"add --interactive", "commit --amend"}
	if !slices.Equal(got, want) {
		t.Fatalf("git %v, want %v", got, want)
	}
}

func TestSaveCommitNonInteractiveTargetOlder(t *testing.T) {
	m := setupSave(t)
	stubUniqueCommits(m, "aaa", "bbb")
	if err := runSaveCommit(prompt.NewNonInteractive(), []string{"bbb"}); err != nil {
		t.Fatal(err)
	}
	got := saveGitActions(m)
	want := []string{
		"add --interactive",
		"commit --fixup=bbb",
		"-c sequence.editor=true rebase --interactive --autosquash --autostash parent",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("git %v, want %v", got, want)
	}
}

func TestSaveCommitNonInteractiveInvalidTarget(t *testing.T) {
	m := setupSave(t)
	stubUniqueCommits(m, "aaa", "bbb")
	err := runSaveCommit(prompt.NewNonInteractive(), []string{"nope"})
	if err == nil || !strings.Contains(err.Error(), "not a valid choice") {
		t.Fatalf("got %v", err)
	}
	if len(saveGitActions(m)) != 0 {
		t.Fatalf("git should not run before invalid target: %v", saveGitActions(m))
	}
}

func TestSaveCommitCLITargetSkipsPicker(t *testing.T) {
	m := setupSave(t)
	stubUniqueCommits(m, "aaa", "bbb")
	p := &sessionPrompter{picks: []string{"should-not-be-used"}}
	if err := runSaveCommit(p, []string{"bbb"}); err != nil {
		t.Fatal(err)
	}
	if len(p.asked) != 0 {
		t.Fatalf("unexpected pick: %v", p.asked)
	}
	got := saveGitActions(m)
	want := []string{
		"add --interactive",
		"commit --fixup=bbb",
		"-c sequence.editor=true rebase --interactive --autosquash --autostash parent",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("git %v, want %v", got, want)
	}
}

func TestSaveCommitParentRevParseFails(t *testing.T) {
	m := setupSave(t)
	stubUniqueCommits(m, "aaa", "bbb")
	m.FailOn["rev-parse bbb^"] = errors.New("bad revision")
	err := runSaveCommit(prompt.NewNonInteractive(), []string{"bbb"})
	if err == nil || !strings.Contains(err.Error(), "cannot find parent of bbb") {
		t.Fatalf("got %v", err)
	}
	got := saveGitActions(m)
	want := []string{"add --interactive"}
	if !slices.Equal(got, want) {
		t.Fatalf("git %v, want %v", got, want)
	}
}

func TestSaveCommitHEADRevParseFailsDoesNotFixup(t *testing.T) {
	m := setupSave(t)
	stubUniqueCommits(m, "aaa", "bbb")
	m.FailOn["rev-parse HEAD"] = errors.New("ambiguous")
	p := &sessionPrompter{picks: []string{"aaa"}}
	err := runSaveCommit(p, nil)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("got %v", err)
	}
	got := saveGitActions(m)
	want := []string{"add --interactive"}
	if !slices.Equal(got, want) {
		t.Fatalf("git %v, want %v", got, want)
	}
}

func TestSaveCommitFixupRebaseFailureResets(t *testing.T) {
	m := setupSave(t)
	stubUniqueCommits(m, "aaa", "bbb")
	m.FailOn["-c sequence.editor=true rebase --interactive --autosquash --autostash parent"] = errors.New("conflict")
	err := runSaveCommit(prompt.NewNonInteractive(), []string{"bbb"})
	if err == nil || err.Error() != "conflict" {
		t.Fatalf("got %v", err)
	}
	got := saveGitActions(m)
	want := []string{
		"add --interactive",
		"commit --fixup=bbb",
		"-c sequence.editor=true rebase --interactive --autosquash --autostash parent",
		"rebase --abort",
		"reset --soft aaa",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("git %v, want %v", got, want)
	}
}

func TestAmendCommandHidden(t *testing.T) {
	c := NewCommand()
	for _, sub := range c.Commands() {
		if sub.Name() == "amend" {
			if !sub.Hidden {
				t.Fatal("work amend should be hidden")
			}
			return
		}
	}
	t.Fatal("hidden work amend shim missing")
}

func setupSave(t *testing.T) *git.MemoryRunner {
	t.Helper()
	m := git.NewMemoryRunner()
	m.Repo.CurrentBranch = "feature"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	return m
}

func stubUniqueCommits(m *git.MemoryRunner, newest, older string) {
	now := saveNowUnix()
	m.Outputs["log --format="+saveCommitLogFormat+" main..feature"] = fmt.Sprintf(
		"%s\t%d\tnewest subject\n%s\t%d\tolder subject",
		newest, now-2*3600, older, now-3*86400,
	)
	m.Outputs["rev-parse HEAD"] = newest
	m.Outputs["rev-parse "+newest] = newest
	m.Outputs["rev-parse "+older] = older
	m.Outputs["rev-parse "+older+"^"] = "parent"
}

func runSaveCommit(p prompt.Prompter, args []string) error {
	var target string
	return saveCommit(saveCmd(p), args, saveSpec(&target))
}

func saveCmd(p prompt.Prompter) *cobra.Command {
	c := &cobra.Command{}
	c.SetContext(prompt.WithPrompter(context.Background(), p))
	return c
}

func choiceValues(choices []prompt.Choice) []string {
	out := make([]string, len(choices))
	for i, c := range choices {
		out[i] = c.Value
	}
	return out
}

func choiceDisplays(choices []prompt.Choice) []string {
	out := make([]string, len(choices))
	for i, c := range choices {
		out[i] = c.Display
	}
	return out
}

func saveGitActions(m *git.MemoryRunner) []string {
	var out []string
	for _, c := range m.Calls {
		if len(c.Args) == 0 {
			continue
		}
		switch c.Args[0] {
		case "add", "commit", "-c", "rebase", "reset":
			out = append(out, strings.Join(c.Args, " "))
		}
	}
	return out
}
