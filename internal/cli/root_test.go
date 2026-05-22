package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootHelpListsAllCommands(t *testing.T) {
	bin := buildTestBinary(t)
	out, err := exec.Command(bin, "--help").Output()
	if err != nil {
		t.Fatalf("--help: %v", err)
	}
	text := string(out)
	for _, name := range []string{
		"acquire-git", "start-work", "show-commands", "release-work",
	} {
		if !strings.Contains(text, name) {
			t.Errorf("help missing command %q", name)
		}
	}
	if !strings.Contains(text, "enable Elegnat Git services") {
		t.Error("help missing service group header")
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

func TestNoWorkflowsFlagAccepted(t *testing.T) {
	bin := buildTestBinary(t)
	cmd := exec.Command(bin, "--no-workflows", "deliver-work")
	out, _ := cmd.CombinedOutput()
	if !strings.Contains(string(out), "not implemented") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func buildTestBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "git-elegant")
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/git-elegant")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}
