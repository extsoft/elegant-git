package repo

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/exitcode"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func TestInitCommandHelpExplainsRefusal(t *testing.T) {
	var buf strings.Builder
	cliruntime.PrintCommandHelp(&buf, newInitCommand())
	out := buf.String()
	for _, want := range []string{
		"init <workspace>",
		"empty initial commit",
		"a git repository",
		"unborn",
		"linked",
		"bare",
		".git",
		"worktree",
		"untracked",
		"hooks",
		"left in place",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q:\n%s", want, out)
		}
	}
}

func TestIsFreshDirectory(t *testing.T) {
	fresh := "fatal: not a git repository (or any of the parent directories): .git"
	cases := []struct {
		name string
		out  string
		err  error
		want bool
	}{
		{"parent search", fresh, fmt.Errorf("exit status 128"), true},
		{"mount boundary", "fatal: not a git repository (or any parent up to mount point /Volumes/ext)\nStopping at filesystem boundary (GIT_DISCOVERY_ACROSS_FILESYSTEM not set).", fmt.Errorf("exit status 128"), true},
		{"explicit git dir", "fatal: not a git repository: '/tmp/missing'", fmt.Errorf("exit status 128"), false},
		{"invalid gitfile", "fatal: invalid gitfile format: /tmp/notgit", fmt.Errorf("exit status 128"), false},
		{"dubious ownership", "fatal: detected dubious ownership in repository at '/tmp/repo'", fmt.Errorf("exit status 128"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isFreshDirectory(tc.out, tc.err); got != tc.want {
				t.Fatalf("isFreshDirectory() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRefuseReinitializationWhenGitContextExists(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	m := git.NewMemoryRunner()
	m.Outputs[repositoryProbe] = "false\ntrue\ntrue"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	buf := captureText(t)
	err := initRun(&cobra.Command{}, "work")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "was not initialized") {
		t.Fatalf("output:\n%s", buf.String())
	}
	for _, c := range m.Calls {
		if len(c.Args) > 0 && c.Args[0] == "init" {
			t.Fatalf("git init ran: %+v", m.Calls)
		}
	}
}

func TestRefuseReinitializationWhenDotGitExists(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(".git", []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := git.NewMemoryRunner()
	m.FailOn[repositoryProbe] = fmt.Errorf("%s .git", freshRepositoryText)
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	err := refuseReinitialization()
	assertWorkflow(t, err)
	if len(m.Calls) != 0 {
		t.Fatalf("git was invoked: %+v", m.Calls)
	}
	body, err := os.ReadFile(".git")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "keep\n" {
		t.Fatalf("body %q", body)
	}
}

func TestRefuseReinitializationReportsUnexpectedDetectionError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	m := git.NewMemoryRunner()
	m.FailOn[repositoryProbe] = fmt.Errorf("fatal: detected dubious ownership in repository")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	buf := captureText(t)
	err := initRun(&cobra.Command{}, "work")
	var es cliruntime.ExitStatus
	if errors.As(err, &es) && es.Code == exitcode.WorkflowError {
		t.Fatalf("unexpected workflow refusal: %v", err)
	}
	if err == nil || !strings.Contains(buf.String(), "dubious ownership") || !strings.Contains(buf.String(), "was not initialized") {
		t.Fatalf("err=%v output:\n%s", err, buf.String())
	}
	for _, c := range m.Calls {
		if len(c.Args) > 0 && c.Args[0] == "init" {
			t.Fatalf("git init ran: %+v", m.Calls)
		}
	}
}

func TestFreshDirectoryAllowsInitializationToStart(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	m := git.NewMemoryRunner()
	m.FailOn[repositoryProbe] = fmt.Errorf("%s .git", freshRepositoryText)
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	if err := refuseReinitialization(); err != nil {
		t.Fatal(err)
	}
}

func TestWriteInitialCommitMessageStaysOutsideTheWorktree(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile("a-message-of-initial-commit", []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err := writeInitialCommitMessage()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	inside, err := pathInside(dir, path)
	if err != nil {
		t.Fatal(err)
	}
	if inside {
		t.Fatalf("message file %s is inside %s", path, dir)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != initialCommitMessage {
		t.Fatalf("message %q", body)
	}
	kept, err := os.ReadFile("a-message-of-initial-commit")
	if err != nil {
		t.Fatal(err)
	}
	if string(kept) != "keep" {
		t.Fatalf("basename %q", kept)
	}
}

func TestWriteInitialCommitMessageRejectsSymlinkedTempDir(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	t.Chdir(real)
	t.Setenv("TMPDIR", link)
	if _, err := writeInitialCommitMessage(); err == nil {
		t.Fatal("expected refusal")
	}
	entries, err := os.ReadDir(real)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("entries %v", names(entries))
	}
}

func TestWriteInitialCommitMessageRefusesWorktreeTempDir(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("TMPDIR", dir)
	if err := os.WriteFile("a-message-of-initial-commit", []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := writeInitialCommitMessage(); err == nil {
		t.Fatal("expected refusal")
	}
	kept, err := os.ReadFile("a-message-of-initial-commit")
	if err != nil {
		t.Fatal(err)
	}
	if string(kept) != "keep" {
		t.Fatalf("basename %q", kept)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "a-message-of-initial-commit" {
		t.Fatalf("entries %+v", names(entries))
	}
}

func names(entries []os.DirEntry) []string {
	out := make([]string, len(entries))
	for i, entry := range entries {
		out[i] = entry.Name()
	}
	return out
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

func assertNotWorkflow(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	var es cliruntime.ExitStatus
	if errors.As(err, &es) && es.Code == exitcode.WorkflowError {
		t.Fatalf("workflow refusal: %v", err)
	}
}
