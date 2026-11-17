package repo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/workflows"
	"github.com/spf13/cobra"
)

func TestRealGitInitRefusesExistingRepository(t *testing.T) {
	setupInitGit(t)
	dir := t.TempDir()
	t.Chdir(dir)
	gitOK(t, dir, "init", "-q")
	gitOK(t, dir, "commit", "--allow-empty", "-m", "base", "-q")
	if err := os.WriteFile("keep.txt", []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("a-message-of-initial-commit", []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	head := gitOK(t, dir, "rev-parse", "HEAD")
	status := gitOK(t, dir, "status", "--porcelain=v1", "-uall")
	buf := captureText(t)
	err := initRun(initCmd(), "work")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "was not initialized") {
		t.Fatalf("output:\n%s", buf.String())
	}
	if gitOK(t, dir, "rev-parse", "HEAD") != head || gitOK(t, dir, "status", "--porcelain=v1", "-uall") != status {
		t.Fatal("repository changed")
	}
	assertFile(t, "keep.txt", "keep\n")
	assertFile(t, "a-message-of-initial-commit", "old\n")
}

func TestRealGitInitRefusesUnbornRepository(t *testing.T) {
	setupInitGit(t)
	dir := t.TempDir()
	t.Chdir(dir)
	gitOK(t, dir, "init", "-q")
	if err := os.WriteFile("keep.txt", []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	status := gitOK(t, dir, "status", "--porcelain=v1", "-uall")
	err := initRun(initCmd(), "work")
	assertWorkflow(t, err)
	if commitExists(dir) {
		t.Fatal("HEAD exists")
	}
	if gitOK(t, dir, "status", "--porcelain=v1", "-uall") != status {
		t.Fatal("status changed")
	}
	assertFile(t, "keep.txt", "keep\n")
}

func TestRealGitInitRefusesLinkedWorktree(t *testing.T) {
	setupInitGit(t)
	parent := t.TempDir()
	linked := t.TempDir()
	gitOK(t, parent, "init", "-q")
	gitOK(t, parent, "commit", "--allow-empty", "-m", "base", "-q")
	gitOK(t, parent, "worktree", "add", "-q", linked, "HEAD")
	t.Chdir(linked)
	before, err := os.ReadFile(".git")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("keep.txt", []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	head := gitOK(t, linked, "rev-parse", "HEAD")
	status := gitOK(t, linked, "status", "--porcelain=v1", "-uall")
	err = initRun(initCmd(), "work")
	assertWorkflow(t, err)
	after, err := os.ReadFile(".git")
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) || gitOK(t, linked, "rev-parse", "HEAD") != head || gitOK(t, linked, "status", "--porcelain=v1", "-uall") != status {
		t.Fatal("linked worktree changed")
	}
	assertFile(t, "keep.txt", "keep\n")
}

func TestRealGitInitRefusesBareRepository(t *testing.T) {
	setupInitGit(t)
	dir := t.TempDir()
	gitOK(t, dir, "init", "-q", "--bare", dir)
	t.Chdir(dir)
	config := gitOK(t, dir, "config", "--local", "--list")
	err := initRun(initCmd(), "work")
	assertWorkflow(t, err)
	if gitOK(t, dir, "config", "--local", "--list") != config || gitOK(t, dir, "rev-parse", "--is-bare-repository") != "true" {
		t.Fatal("bare repository changed")
	}
	if commitExists(dir) {
		t.Fatal("HEAD exists")
	}
}

func TestRealGitInitRefusesBareSubdirectory(t *testing.T) {
	setupInitGit(t)
	dir := t.TempDir()
	gitOK(t, dir, "init", "-q", "--bare", dir)
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "keep.txt"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	err := initRun(initCmd(), "work")
	assertWorkflow(t, err)
	if _, statErr := os.Lstat(".git"); !os.IsNotExist(statErr) {
		t.Fatalf("nested .git: %v", statErr)
	}
	assertFile(t, "keep.txt", "keep\n")
	if gitOK(t, dir, "rev-parse", "--is-bare-repository") != "true" {
		t.Fatal("bare repository changed")
	}
}

func TestRealGitInitRefusesGitFile(t *testing.T) {
	setupInitGit(t)
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(".git", []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := initRun(initCmd(), "work")
	assertWorkflow(t, err)
	info, err := os.Lstat(".git")
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("mode %v", info.Mode())
	}
	assertFile(t, ".git", "keep\n")
}

func TestRealGitInitRefusesNestedWorktree(t *testing.T) {
	setupInitGit(t)
	parent := t.TempDir()
	gitOK(t, parent, "init", "-q")
	gitOK(t, parent, "commit", "--allow-empty", "-m", "base", "-q")
	sub := filepath.Join(parent, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "keep.txt"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	head := gitOK(t, parent, "rev-parse", "HEAD")
	status := gitOK(t, parent, "status", "--porcelain=v1", "-uall")
	t.Chdir(sub)
	err := initRun(initCmd(), "work")
	assertWorkflow(t, err)
	if _, statErr := os.Lstat(filepath.Join(sub, ".git")); !os.IsNotExist(statErr) {
		t.Fatalf("nested .git: %v", statErr)
	}
	if gitOK(t, parent, "rev-parse", "HEAD") != head || gitOK(t, parent, "status", "--porcelain=v1", "-uall") != status {
		t.Fatal("enclosing repository changed")
	}
	assertFile(t, "keep.txt", "keep\n")
}

func TestRealGitInitRefusesBeforeHooksAndPrompts(t *testing.T) {
	setupInitGit(t)
	prev := workflows.Skip
	workflows.Skip = false
	t.Cleanup(func() { workflows.Skip = prev })
	parent := t.TempDir()
	gitOK(t, parent, "init", "-q")
	gitOK(t, parent, "commit", "--allow-empty", "-m", "base", "-q")
	sub := filepath.Join(parent, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(parent, "hookmarker")
	for _, rel := range []struct{ path, word string }{
		{".git/.config/elegant-git/hooks/repo-init-ahead", "ahead"},
		{".git/.config/elegant-git/hooks/repo-init-after", "after"},
		{".git/.workflows/acquire-repository-ahead", "legacy-ahead"},
		{".git/.workflows/acquire-repository-after", "legacy-after"},
	} {
		writeInitHook(t, parent, rel.path, marker, rel.word)
	}
	head := gitOK(t, parent, "rev-parse", "HEAD")
	status := gitOK(t, parent, "status", "--porcelain=v1", "-uall")
	t.Chdir(sub)
	buf := captureText(t)
	cmd := newInitCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	cmd.SetArgs([]string{"work"})
	err := cmd.Execute()
	assertWorkflow(t, err)
	out := buf.String()
	if !strings.Contains(out, "was not initialized") || strings.Contains(out, "is not a valid choice") {
		t.Fatalf("err=%v output:\n%s", err, out)
	}
	if _, statErr := os.Lstat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("hook marker: %v", statErr)
	}
	if _, statErr := os.Lstat(filepath.Join(sub, ".git")); !os.IsNotExist(statErr) {
		t.Fatalf("nested .git: %v", statErr)
	}
	if gitOK(t, parent, "rev-parse", "HEAD") != head || gitOK(t, parent, "status", "--porcelain=v1", "-uall") != status {
		t.Fatal("enclosing repository changed")
	}
}

func writeInitHook(t *testing.T, root, rel, marker, word string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf("#!/bin/bash\nprintf '%%s\\n' %s >> %s\n", strconv.Quote(word), strconv.Quote(marker))
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestRealGitInitReportsInvalidGitDir(t *testing.T) {
	setupInitGit(t)
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile("notgit", []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_DIR", filepath.Join(dir, "notgit"))
	buf := captureText(t)
	err := initRun(initCmd(), "work")
	assertNotWorkflow(t, err)
	if !strings.Contains(buf.String(), "was not initialized") {
		t.Fatalf("err=%v output:\n%s", err, buf.String())
	}
	if _, statErr := os.Lstat(".git"); !os.IsNotExist(statErr) {
		t.Fatalf(".git: %v", statErr)
	}
	info, err := os.Lstat("notgit")
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("mode %v", info.Mode())
	}
	assertFile(t, "notgit", "x\n")
}

func TestRealGitInitReportsMissingGitDir(t *testing.T) {
	setupInitGit(t)
	dir := t.TempDir()
	t.Chdir(dir)
	missing := filepath.Join(dir, "missing")
	t.Setenv("GIT_DIR", missing)
	err := initRun(initCmd(), "work")
	assertNotWorkflow(t, err)
	if _, statErr := os.Lstat(missing); !os.IsNotExist(statErr) {
		t.Fatalf("missing git dir was created: %v", statErr)
	}
	if _, statErr := os.Lstat(".git"); !os.IsNotExist(statErr) {
		t.Fatalf(".git: %v", statErr)
	}
}

func TestRealGitInitPreservesFilesAndWritesEmptyCommit(t *testing.T) {
	setupInitGit(t)
	saveWorkspace(t, "work", "Init User", "init@example.com")
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile("notes.txt", []byte("notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("a-message-of-initial-commit", []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := initRun(initCmd(), "work"); err != nil {
		t.Fatal(err)
	}
	assertFile(t, "notes.txt", "notes\n")
	assertFile(t, "a-message-of-initial-commit", "old\n")
	if gitOK(t, dir, "ls-files") != "" {
		t.Fatalf("staged files:\n%s", gitOK(t, dir, "ls-files"))
	}
	status := gitOK(t, dir, "status", "--porcelain=v1", "-uall")
	if !strings.Contains(status, "?? notes.txt") || !strings.Contains(status, "?? a-message-of-initial-commit") {
		t.Fatalf("status:\n%s", status)
	}
	tree := gitOK(t, dir, "rev-parse", "HEAD^{tree}")
	empty := gitStdin(t, dir, strings.NewReader(""), "mktree")
	if tree != empty || gitOK(t, dir, "ls-tree", "-r", "HEAD") != "" {
		t.Fatalf("tree %s ls-tree %q", tree, gitOK(t, dir, "ls-tree", "-r", "HEAD"))
	}
	if strings.TrimRight(gitRaw(t, dir, "log", "-1", "--format=%B"), "\n") != strings.TrimRight(initialCommitMessage, "\n") {
		t.Fatalf("message %q", gitRaw(t, dir, "log", "-1", "--format=%B"))
	}
	if gitOK(t, dir, "config", "--local", "--get", "user.name") != "Init User" {
		t.Fatalf("user.name %s", gitOK(t, dir, "config", "--local", "--get", "user.name"))
	}
	gitDir, err := memrepo.GitDir()
	if err != nil {
		t.Fatal(err)
	}
	state, err := memrepo.Load(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	if state.DefaultBranch != "main" || len(state.ProtectedBranches) != 1 || state.ProtectedBranches[0] != "main" || state.WorkspaceID == "" {
		t.Fatalf("state %+v", state)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := names(entries)
	want := map[string]bool{".git": true, "notes.txt": true, "a-message-of-initial-commit": true}
	if len(got) != len(want) {
		t.Fatalf("entries %v", got)
	}
	for _, name := range got {
		if !want[name] {
			t.Fatalf("unexpected %s in %v", name, got)
		}
	}
}

func TestRealGitInitLeavesRepositoryWhenConfigurationFails(t *testing.T) {
	setupInitGit(t)
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile("notes.txt", []byte("notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("a-message-of-initial-commit", []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	buf := captureText(t)
	err := initRun(initCmd(), "missing")
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("err=%v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"Initialized a git repository.",
		"Configuration did not finish, so the initial commit was not created.",
		"The new repository and its files were left in place.",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
	if _, statErr := os.Lstat(".git"); statErr != nil {
		t.Fatal(statErr)
	}
	if commitExists(dir) {
		t.Fatal("HEAD exists")
	}
	assertFile(t, "notes.txt", "notes\n")
	assertFile(t, "a-message-of-initial-commit", "old\n")
	if gitOK(t, dir, "ls-files") != "" {
		t.Fatalf("staged:\n%s", gitOK(t, dir, "ls-files"))
	}
}

func TestRealGitInitRefusesStagedIndex(t *testing.T) {
	setupInitGit(t)
	dir := t.TempDir()
	t.Chdir(dir)
	gitOK(t, dir, "init", "-q")
	if err := os.WriteFile("keep.txt", []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOK(t, dir, "add", "--", "keep.txt")
	buf := captureText(t)
	err := commitEmptyInitial()
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "left as they are") {
		t.Fatalf("output:\n%s", buf.String())
	}
	if commitExists(dir) {
		t.Fatal("HEAD exists")
	}
	if gitOK(t, dir, "show", ":keep.txt") != "keep" {
		t.Fatalf("index %q", gitOK(t, dir, "show", ":keep.txt"))
	}
	assertFile(t, "keep.txt", "keep\n")
}

func TestRealGitInitRefusesIndexStagedBeforeHeadMoves(t *testing.T) {
	setupInitGit(t)
	dir := t.TempDir()
	t.Chdir(dir)
	gitOK(t, dir, "init", "-q")
	if err := os.WriteFile("keep.txt", []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := writeEmptyInitialCommit(func() error {
		gitOK(t, dir, "add", "--", "keep.txt")
		return nil
	})
	assertWorkflow(t, err)
	if commitExists(dir) {
		t.Fatal("HEAD exists")
	}
	if gitOK(t, dir, "show", ":keep.txt") != "keep" {
		t.Fatalf("index %q", gitOK(t, dir, "show", ":keep.txt"))
	}
	assertFile(t, "keep.txt", "keep\n")
}

func setupInitGit(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"GIT_DIR",
		"GIT_WORK_TREE",
		"GIT_INDEX_FILE",
		"GIT_COMMON_DIR",
		"GIT_PREFIX",
		"GIT_OBJECT_DIRECTORY",
		"GIT_ALTERNATE_OBJECT_DIRECTORIES",
		"GIT_NAMESPACE",
	} {
		unsetenv(t, key)
	}
	cfg := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(cfg, []byte("[init]\n\tdefaultBranch = main\n[advice]\n\tdefaultBranchName = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", cfg)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(t.TempDir(), "state.json"))
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@example.com")
	t.Setenv("GIT_PAGER", "cat")
	t.Setenv("PAGER", "cat")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
	git.Use(git.RealRunner{})
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
}

func unsetenv(t *testing.T, key string) {
	t.Helper()
	prev, ok := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(key, prev)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func saveWorkspace(t *testing.T, name, user, email string) {
	t.Helper()
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: name, UserName: user, UserEmail: email,
	}); err != nil {
		t.Fatal(err)
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
}

func initCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	return cmd
}

func assertFile(t *testing.T, name, body string) {
	t.Helper()
	got, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("%s %q", name, got)
	}
}

func gitOK(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := gitCombined(dir, args...)
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(out)
}

func gitRaw(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := gitCombined(dir, args...)
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

func commitExists(dir string) bool {
	_, ok := gitRun(dir, "rev-parse", "--verify", "--quiet", "HEAD")
	return ok
}

func gitRun(dir string, args ...string) (string, bool) {
	out, err := gitCombined(dir, args...)
	if err != nil {
		return strings.TrimSpace(out), false
	}
	return strings.TrimSpace(out), true
}

func gitStdin(t *testing.T, dir string, in *strings.Reader, args ...string) string {
	t.Helper()
	cmd := exec.Command(git.Executable(), args...)
	cmd.Dir = dir
	cmd.Stdin = in
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func gitCombined(dir string, args ...string) (string, error) {
	cmd := exec.Command(git.Executable(), args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
