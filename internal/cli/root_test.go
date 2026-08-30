package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootHelpListsObjects(t *testing.T) {
	bin := buildTestBinary(t)
	out, err := exec.Command(bin, "--help").Output()
	if err != nil {
		t.Fatalf("--help: %v", err)
	}
	text := string(out)
	for _, want := range []string{"git configure", "git list", "repo clone", "memory workspaces", "work start", "hook list", "release new"} {
		if !strings.Contains(text, want) {
			t.Errorf("help missing %q", want)
		}
	}
	if !strings.Contains(text, "Objects:") {
		t.Error("help missing Objects section")
	}
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
	cmd := exec.Command(bin, "memory", "workspaces", "--unknown-flag")
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
	if strings.Contains(text, "Objects:") && strings.Contains(text, "  git —") {
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
