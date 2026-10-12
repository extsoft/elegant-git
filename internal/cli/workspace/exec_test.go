package workspace

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

func TestMain(m *testing.M) {
	if os.Getenv("EG_EXEC_HELPER") == "1" {
		os.Exit(runExecHelper())
	}
	os.Exit(m.Run())
}

func runExecHelper() int {
	args := helperArgs()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "helper: missing mode")
		return 2
	}
	switch args[0] {
	case "print-args":
		fmt.Print(strings.Join(args[1:], "\x1e"))
		return 0
	case "pwd":
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Print(wd)
		return 0
	case "env":
		for _, key := range []string{"EG_WORKSPACE", "EG_REPO_NAME", "EG_REPO_PATH"} {
			fmt.Printf("%s=%s\n", key, os.Getenv(key))
		}
		return 0
	case "cat":
		_, err := io.Copy(os.Stdout, os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "streams":
		const chunk = 4096
		outDone := make(chan struct{})
		errDone := make(chan struct{})
		go func() {
			for i := 0; i < 32; i++ {
				if _, err := os.Stdout.Write(bytes.Repeat([]byte("O"), chunk)); err != nil {
					break
				}
			}
			close(outDone)
		}()
		go func() {
			for i := 0; i < 32; i++ {
				if _, err := os.Stderr.Write(bytes.Repeat([]byte("E"), chunk)); err != nil {
					break
				}
			}
			close(errDone)
		}()
		<-outDone
		<-errDone
		return 0
	case "exit":
		if len(args) < 2 {
			return 2
		}
		var code int
		if _, err := fmt.Sscanf(args[1], "%d", &code); err != nil {
			return 2
		}
		return code
	default:
		fmt.Fprintf(os.Stderr, "helper: unknown mode %q\n", args[0])
		return 2
	}
}

func helperArgs() []string {
	for i, arg := range os.Args {
		if arg == "--" && i+1 < len(os.Args) {
			return os.Args[i+1:]
		}
	}
	return nil
}

func helperArgv(mode string, args ...string) []string {
	cmdArgs := []string{"-test.run=TestExecHelperProcess", "--", mode}
	cmdArgs = append(cmdArgs, args...)
	return append([]string{os.Args[0]}, cmdArgs...)
}

func TestExecHelperProcess(t *testing.T) {
	if os.Getenv("EG_EXEC_HELPER") != "1" {
		t.Skip("helper subprocess only")
	}
	os.Exit(runExecHelper())
}

func setupExecWorkspace(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	t.Setenv("EG_EXEC_HELPER", "1")
	repoA := filepath.Join(dir, "a")
	repoB := filepath.Join(dir, "b")
	for _, p := range []string{repoA, repoB} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{ID: "ra", Name: "a", WorkspaceID: id, CurrentPath: repoA})
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{ID: "rb", Name: "b", WorkspaceID: id, CurrentPath: repoB})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	return repoA, repoB, "acme"
}

func TestExecRunUsesRepoDirectoryAndEnv(t *testing.T) {
	repoA, repoB, ws := setupExecWorkspace(t)
	var stdout, stderr bytes.Buffer
	err := execRun(execStreams{
		Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr,
	}, ws, helperArgv("env"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "==> a ("+repoA+")") {
		t.Fatalf("stderr:\n%s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "==> b ("+repoB+")") {
		t.Fatalf("stderr:\n%s", stderr.String())
	}
	for _, want := range []string{"EG_WORKSPACE=acme", "EG_REPO_NAME=a", "EG_REPO_PATH=" + repoA} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("missing %q in stdout:\n%s", want, stdout.String())
		}
	}
}

func TestExecRunStdinAndSeparateStreams(t *testing.T) {
	_, _, ws := setupExecWorkspace(t)
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- execRun(execStreams{
			Stdin: strings.NewReader("pipe-in"), Stdout: stdoutW, Stderr: stderrW,
		}, ws, helperArgv("cat"))
	}()
	var outBuf, errBuf bytes.Buffer
	outCopy := make(chan error, 1)
	errCopy := make(chan error, 1)
	go func() { _, err := io.Copy(&outBuf, stdoutR); outCopy <- err }()
	go func() { _, err := io.Copy(&errBuf, stderrR); errCopy <- err }()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	_ = stdoutW.Close()
	_ = stderrW.Close()
	if err := <-outCopy; err != nil {
		t.Fatal(err)
	}
	if err := <-errCopy; err != nil {
		t.Fatal(err)
	}
	if outBuf.String() != "pipe-in" {
		t.Fatalf("stdout=%q", outBuf.String())
	}
	if !strings.Contains(errBuf.String(), "==> a (") || !strings.Contains(errBuf.String(), "==> b (") {
		t.Fatalf("stderr headers missing:\n%s", errBuf.String())
	}
}

func TestExecRunLargeStreamsNoDeadlock(t *testing.T) {
	_, _, ws := setupExecWorkspace(t)
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- execRun(execStreams{
			Stdin: strings.NewReader(""), Stdout: stdoutW, Stderr: stderrW,
		}, ws, helperArgv("streams"))
	}()
	var outBuf, errBuf bytes.Buffer
	outCopy := make(chan error, 1)
	errCopy := make(chan error, 1)
	go func() { _, err := io.Copy(&outBuf, stdoutR); outCopy <- err }()
	go func() { _, err := io.Copy(&errBuf, stderrR); errCopy <- err }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for execRun")
	}
	_ = stdoutW.Close()
	_ = stderrW.Close()
	if err := <-outCopy; err != nil {
		t.Fatal(err)
	}
	if err := <-errCopy; err != nil {
		t.Fatal(err)
	}
	if len(outBuf.String()) < 32*4096*2 {
		t.Fatalf("stdout too short: %d", len(outBuf.String()))
	}
	if len(errBuf.String()) < 32*4096*2 {
		t.Fatalf("stderr too short: %d", len(errBuf.String()))
	}
}

func TestExecRunFailFast(t *testing.T) {
	_, _, ws := setupExecWorkspace(t)
	var stderr bytes.Buffer
	err := execRun(execStreams{
		Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &stderr,
	}, ws, helperArgv("exit", "7"))
	var es cliruntime.ExitStatus
	if !errors.As(err, &es) || es.Code != 7 {
		t.Fatalf("err=%v", err)
	}
	if strings.Count(stderr.String(), "==> ") != 1 {
		t.Fatalf("expected one repo header, got:\n%s", stderr.String())
	}
}

func TestExecRunMissingExecutable(t *testing.T) {
	_, _, ws := setupExecWorkspace(t)
	var stderr bytes.Buffer
	err := execRun(execStreams{
		Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &stderr,
	}, ws, []string{"eg-workspace-exec-no-such-binary-" + t.Name()})
	var es cliruntime.ExitStatus
	if !errors.As(err, &es) || es.Code != 127 {
		t.Fatalf("err=%v", err)
	}
}

func TestExecRunMissingPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	missing := filepath.Join(dir, "gone")
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "ra", Name: "a", WorkspaceID: id, CurrentPath: missing,
	})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	err = execRun(execStreams{
		Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &stderr,
	}, "acme", []string{"true"})
	if err == nil {
		t.Fatal("expected error")
	}
	out := stderr.String()
	for _, want := range []string{
		"path missing: " + missing,
		"Run `eg workspace doctor acme` to repair missing paths from the workspace side,",
		"or `cd` into a relocated clone and run `eg repo doctor`.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestExecRunNoLinkedRepos(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	}); err != nil {
		t.Fatal(err)
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if err := execRun(execStreams{
		Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &stderr,
	}, "acme", []string{"true"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "No repositories linked to workspace acme") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestExecCommandRequiresDash(t *testing.T) {
	cmd := newExecCommand()
	cmd.SetContext(prompt.WithPrompter(t.Context(), prompt.NewNonInteractive()))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.RunE(cmd, []string{"git", "status"})
	if !cliruntime.IsUsageError(err) {
		t.Fatalf("err=%v", err)
	}
}

func TestExecCommandRequiresChild(t *testing.T) {
	root := &cobra.Command{Use: "eg"}
	root.AddCommand(newExecCommand())
	root.SetContext(prompt.WithPrompter(t.Context(), prompt.NewNonInteractive()))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"exec", "--"})
	if err := root.Execute(); err == nil || !cliruntime.IsUsageError(err) {
		t.Fatalf("err=%v", err)
	}
}

func TestExecCommandAcceptsArgsAfterDash(t *testing.T) {
	_, _, ws := setupExecWorkspace(t)
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	os.Stdout = stdoutW
	t.Cleanup(func() {
		os.Stdout = oldStdout
		_ = stdoutW.Close()
	})

	root := &cobra.Command{Use: "eg"}
	root.AddCommand(newExecCommand())
	root.SetContext(prompt.WithPrompter(t.Context(), prompt.NewNonInteractive()))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	args := append([]string{"exec", ws, "--"}, helperArgv("print-args", "ok")...)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	_ = stdoutW.Close()
	var stdout bytes.Buffer
	if _, err := io.Copy(&stdout, stdoutR); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "okok" {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestExecCommandMissingDashIsUsage(t *testing.T) {
	root := &cobra.Command{Use: "eg"}
	root.AddCommand(newExecCommand())
	root.SetContext(prompt.WithPrompter(t.Context(), prompt.NewNonInteractive()))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"exec", "git", "status"})
	err := root.Execute()
	if err == nil || !cliruntime.IsUsageError(err) || !strings.Contains(err.Error(), "exec requires -- before the command") {
		t.Fatalf("err=%v", err)
	}
}

func TestExecCommandRejectsTwoNamesBeforeDash(t *testing.T) {
	root := &cobra.Command{Use: "eg"}
	root.AddCommand(newExecCommand())
	root.SetContext(prompt.WithPrompter(t.Context(), prompt.NewNonInteractive()))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"exec", "a", "b", "--", "true"})
	err := root.Execute()
	if err == nil || !cliruntime.IsUsageError(err) {
		t.Fatalf("err=%v", err)
	}
}

func TestExecCommandBareInteractive(t *testing.T) {
	var ran bool
	orig := runExecInteractiveFn
	runExecInteractiveFn = func(*cobra.Command) error {
		ran = true
		return nil
	}
	defer func() { runExecInteractiveFn = orig }()

	root := &cobra.Command{Use: "eg"}
	root.AddCommand(newExecCommand())
	root.SetContext(prompt.WithPrompter(t.Context(), prompt.NewNonInteractive()))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"exec"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("expected interactive exec handler")
	}
}

func TestExecCommandBareNonInteractive(t *testing.T) {
	root := &cobra.Command{Use: "eg"}
	root.AddCommand(newExecCommand())
	root.SetContext(prompt.WithPrompter(t.Context(), prompt.NewNonInteractive()))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"exec"})
	err := root.Execute()
	if err == nil || !cliruntime.IsUsageError(err) {
		t.Fatalf("err=%v", err)
	}
}

func TestExecRunRepoDirectory(t *testing.T) {
	repoA, repoB, ws := setupExecWorkspace(t)
	repoAR, err := filepath.EvalSymlinks(repoA)
	if err != nil {
		repoAR = repoA
	}
	repoBR, err := filepath.EvalSymlinks(repoB)
	if err != nil {
		repoBR = repoB
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := execRun(execStreams{
		Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr,
	}, ws, helperArgv("pwd")); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != repoAR+repoBR {
		t.Fatalf("stdout=%q want %q", stdout.String(), repoAR+repoBR)
	}
}

func TestExecRunEnvVars(t *testing.T) {
	repoA, _, ws := setupExecWorkspace(t)
	var stdout bytes.Buffer
	if err := execRun(execStreams{
		Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: io.Discard,
	}, ws, helperArgv("env")); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"EG_WORKSPACE=acme",
		"EG_REPO_NAME=a",
		"EG_REPO_PATH=" + repoA,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("missing %q in %q", want, stdout.String())
		}
	}
}

func TestExecRunArgsVerbatimIntegration(t *testing.T) {
	_, _, ws := setupExecWorkspace(t)
	var stdout bytes.Buffer
	if err := execRun(execStreams{
		Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: io.Discard,
	}, ws, helperArgv("print-args", " spaced ", "$X", "*", "-n")); err != nil {
		t.Fatal(err)
	}
	want := " spaced \x1e$X\x1e*\x1e-n"
	if got := stdout.String(); got != want+want {
		t.Fatalf("stdout=%q", got)
	}
}

func TestParseExecLine(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want []string
	}{
		{"git status -s", []string{"git", "status", "-s"}},
		{`sh -c 'echo hi'`, []string{"sh", "-c", "echo hi"}},
		{`echo "a b"`, []string{"echo", "a b"}},
		{`git commit -m ""`, []string{"git", "commit", "-m", ""}},
		{`cmd ''`, []string{"cmd", ""}},
	} {
		got, err := parseExecLine(tc.in)
		if err != nil {
			t.Fatalf("parseExecLine(%q): %v", tc.in, err)
		}
		if len(got) != len(tc.want) {
			t.Fatalf("parseExecLine(%q)=%v want %v", tc.in, got, tc.want)
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Fatalf("parseExecLine(%q)=%v want %v", tc.in, got, tc.want)
			}
		}
	}
}

func TestRunChildNotExecutable(t *testing.T) {
	dir := t.TempDir()
	rel := filepath.Join(dir, "tool")
	if err := os.WriteFile(rel, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, err := runChild([]string{"./tool"}, dir, nil, execStreams{
		Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	if code != 126 {
		t.Fatalf("relative code=%d", code)
	}

	bin := t.TempDir()
	name := "eg-exec-not-executable"
	if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	code, err = runChild([]string{name}, dir, nil, execStreams{
		Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	if code != 126 {
		t.Fatalf("path code=%d", code)
	}
}

func TestWatchChildForwardsSIGTERM(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix signal semantics")
	}
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	stop := watchChildSignals(cmd)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	time.Sleep(50 * time.Millisecond)
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case waitErr := <-done:
		code, err := mapChildExit("sleep", waitErr)
		if err != nil {
			t.Fatal(err)
		}
		if code != 143 {
			t.Fatalf("code=%d", code)
		}
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("timed out waiting for forwarded SIGTERM")
	}
}

func TestMapChildExitSignalUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix signal semantics")
	}
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := syscall.Kill(cmd.Process.Pid, syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	waitErr := cmd.Wait()
	code, err := mapChildExit("sleep", waitErr)
	if err != nil {
		t.Fatal(err)
	}
	if code != 143 {
		t.Fatalf("code=%d want 143", code)
	}
}
