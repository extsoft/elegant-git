package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/spf13/cobra"
)

func TestGitInformationalSetIsExact(t *testing.T) {
	if gitInformational(&cobra.Command{Use: "completion-extra"}) {
		t.Fatal("command-name substring matched")
	}
	if gitInformational(&cobra.Command{Use: "doctor"}) {
		t.Fatal("doctor is not an informational exception")
	}
	for _, name := range []string{"help", "version", "completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd} {
		if !gitInformational(&cobra.Command{Use: name}) {
			t.Fatalf("%s should stay available", name)
		}
	}
}

func TestUnsupportedGitRefusesBeforeSideEffects(t *testing.T) {
	restore := installOldGitProbe(t)
	defer restore()
	state := filepath.Join(t.TempDir(), "state.json")
	t.Setenv("ELEGANT_GIT_STATE_FILE", state)
	t.Setenv("ELEGANT_GIT_DEPTH", "")

	repo := initBareWorktree(t)
	journal := filepath.Join(repo, ".git", "elegant-git", "recovery.json")
	if err := os.MkdirAll(filepath.Dir(journal), 0o755); err != nil {
		t.Fatal(err)
	}
	const pending = "pending-recovery\n"
	if err := os.WriteFile(journal, []byte(pending), 0o644); err != nil {
		t.Fatal(err)
	}
	before := gitStatus(t, repo)
	t.Chdir(repo)

	for _, args := range [][]string{{"work", "start"}, {"show-commands"}, {"repo", "prune"}} {
		cmd, _, err := rootCmd.Find(args)
		if err != nil {
			t.Fatal(err)
		}
		err = rootCmd.PersistentPreRunE(cmd, nil)
		var usage *cliruntime.UsageError
		if err == nil || errors.As(err, &usage) || !strings.Contains(err.Error(), git.MinVersion) {
			t.Fatalf("%v: %v", args, err)
		}
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatalf("shared memory written: %v", err)
	}
	got, err := os.ReadFile(journal)
	if err != nil || string(got) != pending {
		t.Fatalf("journal %q err=%v", got, err)
	}
	if after := gitStatus(t, repo); after != before {
		t.Fatalf("status changed\nbefore %q\nafter %q", before, after)
	}
}

func TestInformationalCommandsSkipGitAndMigrations(t *testing.T) {
	git.SetVersionProbeForTest(func(string) (string, error) {
		t.Fatal("informational command probed git")
		return "", nil
	})
	t.Cleanup(func() {
		git.SetVersionProbeForTest(nil)
		git.Use(git.RealRunner{})
	})
	state := filepath.Join(t.TempDir(), "state.json")
	t.Setenv("ELEGANT_GIT_STATE_FILE", state)
	t.Setenv("ELEGANT_GIT_DEPTH", "")

	commands := []*cobra.Command{
		mustFind(t, "version"),
		mustFind(t, "completion"),
		{Use: "help"},
		{Use: cobra.ShellCompRequestCmd},
	}
	work := mustFind(t, "work", "start")
	work.InitDefaultHelpFlag()
	if err := work.Flags().Set("help", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = work.Flags().Set("help", "false") })
	commands = append(commands, work)

	rootCmd.InitDefaultVersionFlag()
	if err := rootCmd.Flags().Set("version", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rootCmd.Flags().Set("version", "false") })
	commands = append(commands, mustFind(t, "work", "sync"))

	for _, cmd := range commands {
		if err := rootCmd.PersistentPreRunE(cmd, nil); err != nil {
			t.Fatalf("%s: %v", cmd.Name(), err)
		}
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatalf("shared memory written: %v", err)
	}
}

func TestSelfDoctorSkipsSideEffectsWhenGitIsOld(t *testing.T) {
	restore := installOldGitProbe(t)
	defer restore()
	state := filepath.Join(t.TempDir(), "state.json")
	t.Setenv("ELEGANT_GIT_STATE_FILE", state)
	t.Setenv("ELEGANT_GIT_DEPTH", "")
	for _, args := range [][]string{{"self", "doctor"}, {"git", "doctor"}} {
		cmd := mustFind(t, args...)
		if err := rootCmd.PersistentPreRunE(cmd, nil); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatalf("shared memory written: %v", err)
	}
}

func installOldGitProbe(t *testing.T) func() {
	t.Helper()
	git.SetVersionProbeForTest(func(exe string) (string, error) {
		if exe != git.Executable() {
			t.Fatalf("probed %q, selected %q", exe, git.Executable())
		}
		return "git version 2.39.0", nil
	})
	return func() {
		git.SetVersionProbeForTest(nil)
		git.Use(git.RealRunner{})
	}
}

func mustFind(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	cmd, _, err := rootCmd.Find(args)
	if err != nil {
		t.Fatal(err)
	}
	return cmd
}

func initBareWorktree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command(git.Executable(), "init", "-q", "-b", "main", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	return dir
}

func gitStatus(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command(git.Executable(), "status", "--porcelain=v1")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	return string(out)
}
