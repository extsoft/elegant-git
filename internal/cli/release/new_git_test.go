package release

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/exitcode"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/text"
)

func TestReleaseTagMessageStaysOutOfTheWorktree(t *testing.T) {
	dir := t.TempDir()
	gitCmd(t, dir, "init", "-b", "main")
	gitCmd(t, dir, "config", "user.email", "dev@example.com")
	gitCmd(t, dir, "config", "user.name", "Dev")
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("o\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, dir, "add", "f")
	gitCmd(t, dir, "commit", "-q", "-m", "o")
	bare := t.TempDir()
	gitCmd(t, bare, "init", "--bare")
	gitCmd(t, dir, "remote", "add", "origin", bare)
	gitCmd(t, dir, "push", "-u", "origin", "main")
	if err := os.WriteFile(filepath.Join(dir, "tag-message"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, dir, "add", "tag-message")
	gitCmd(t, dir, "commit", "-q", "-m", "keep")
	gitCmd(t, dir, "push", "origin", "main")
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	git.Use(git.RealRunner{})
	t.Setenv("GIT_EDITOR", "true")
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(dir, ".git", "elegant-git", "state.json"))
	cmd := newNewCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := newRun(cmd, []string{"v1"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "tag-message"))
	if err != nil || string(body) != "keep\n" {
		t.Fatalf("tag-message %q err=%v", body, err)
	}
	if !strings.Contains(gitCmd(t, dir, "tag", "--list"), "v1") {
		t.Fatal("tag missing")
	}
}

func TestReleaseNoRemoteRefusesIgnoredOnDefaultBranch(t *testing.T) {
	dir := t.TempDir()
	gitCmd(t, dir, "init", "-b", "main")
	gitCmd(t, dir, "config", "user.email", "dev@example.com")
	gitCmd(t, dir, "config", "user.name", "Dev")
	if err := os.MkdirAll(filepath.Join(dir, "vendor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vendor", "lib.txt"), []byte("theirs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, dir, "add", "vendor/lib.txt", "f")
	gitCmd(t, dir, "commit", "-q", "-m", "base")
	gitCmd(t, dir, "checkout", "-q", "-b", "feature")
	gitCmd(t, dir, "rm", "-q", "vendor/lib.txt")
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("vendor/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, dir, "add", ".gitignore")
	gitCmd(t, dir, "commit", "-q", "-m", "untrack vendor")
	if err := os.MkdirAll(filepath.Join(dir, "vendor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vendor", "lib.txt"), []byte("MINE ignored\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	git.Use(git.RealRunner{})
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(dir, ".git", "elegant-git", "state.json"))
	var buf bytes.Buffer
	text.SetOutput(&buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	cmd := newNewCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := cmd.Flags().Set("dirty", "discard"); err != nil {
		t.Fatal(err)
	}
	err = newRun(cmd, []string{"v1"})
	var status cliruntime.ExitStatus
	if !errors.As(err, &status) || status.Code != exitcode.WorkflowError {
		t.Fatalf("err=%v out=%q", err, buf.String())
	}
	if !strings.Contains(buf.String(), "vendor/lib.txt would be overwritten") {
		t.Fatalf("out=%q", buf.String())
	}
	if strings.TrimSpace(gitCmd(t, dir, "rev-parse", "--abbrev-ref", "HEAD")) != "feature" {
		t.Fatalf("head %s", gitCmd(t, dir, "rev-parse", "--abbrev-ref", "HEAD"))
	}
	body, err := os.ReadFile(filepath.Join(dir, "vendor", "lib.txt"))
	if err != nil || string(body) != "MINE ignored\n" {
		t.Fatalf("vendor %q err=%v", body, err)
	}
	tracked, err := os.ReadFile(filepath.Join(dir, "f"))
	if err != nil || string(tracked) != "dirty\n" {
		t.Fatalf("f %q err=%v", tracked, err)
	}
	if strings.TrimSpace(gitCmd(t, dir, "tag", "--list")) != "" {
		t.Fatal("tag was created")
	}
}

func gitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command(git.Executable(), args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_EMAIL=dev@example.com", "GIT_AUTHOR_NAME=Dev", "GIT_COMMITTER_EMAIL=dev@example.com", "GIT_COMMITTER_NAME=Dev")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}
