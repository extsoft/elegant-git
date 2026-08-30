package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyProfileAliasRecordsDeprecation(t *testing.T) {
	bin := buildTestBinary(t)
	dir := t.TempDir()
	state := filepath.Join(dir, "state.json")
	if err := os.WriteFile(state, []byte(`{"schema_version":2,"workspaces":{},"repositories":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "profile", "status")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "ELEGANT_GIT_STATE_FILE="+state)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_ = cmd.Run()
	if !strings.Contains(stderr.String(), "deprecated") || !strings.Contains(stderr.String(), "profile") {
		t.Fatalf("expected profile deprecation warning, stderr=%q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "workspace") {
		t.Fatalf("expected workspace replacement hint, stderr=%q", stderr.String())
	}
}

func TestLegacyWorkspaceStatusAliasRecordsDeprecation(t *testing.T) {
	bin := buildTestBinary(t)
	dir := t.TempDir()
	state := filepath.Join(dir, "state.json")
	if err := os.WriteFile(state, []byte(`{"schema_version":2,"workspaces":{},"repositories":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "workspace", "status")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "ELEGANT_GIT_STATE_FILE="+state)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_ = cmd.Run()
	if !strings.Contains(stderr.String(), "deprecated") || !strings.Contains(stderr.String(), "workspace status") {
		t.Fatalf("expected workspace status deprecation warning, stderr=%q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "workspace list current") {
		t.Fatalf("expected workspace list current replacement hint, stderr=%q", stderr.String())
	}
}

func TestLegacyWorkspaceCreateAliasRecordsDeprecation(t *testing.T) {
	bin := buildTestBinary(t)
	dir := t.TempDir()
	state := filepath.Join(dir, "state.json")
	if err := os.WriteFile(state, []byte(`{"schema_version":2,"workspaces":{},"repositories":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "--non-interactive", "workspace", "create", "dz", "D", "d@x.com")
	cmd.Env = append(os.Environ(), "ELEGANT_GIT_STATE_FILE="+state)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_ = cmd.Run()
	if !strings.Contains(stderr.String(), "deprecated") || !strings.Contains(stderr.String(), "workspace create") {
		t.Fatalf("expected workspace create deprecation warning, stderr=%q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "workspace new") {
		t.Fatalf("expected workspace new replacement hint, stderr=%q", stderr.String())
	}
}

func TestLegacyMemoryProfilesAliasParsesFormat(t *testing.T) {
	bin := buildTestBinary(t)
	dir := t.TempDir()
	state := filepath.Join(dir, "state.json")
	if err := os.WriteFile(state, []byte(`{"schema_version":2,"workspaces":{},"repositories":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "memory", "profiles", "--format=json")
	cmd.Env = append(os.Environ(), "ELEGANT_GIT_STATE_FILE="+state)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "{") {
		t.Fatalf("expected json output, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "deprecated") || !strings.Contains(stderr.String(), "memory profiles") {
		t.Fatalf("expected memory profiles deprecation warning, stderr=%q", stderr.String())
	}
}

func TestLegacyStatusAliasRecordsDeprecation(t *testing.T) {
	bin := buildTestBinary(t)
	dir := t.TempDir()
	state := filepath.Join(dir, "state.json")
	if err := os.WriteFile(state, []byte(`{"schema_version":2,"workspaces":{},"repositories":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		object      string
		replacement string
	}{
		{"memory", "memory list"},
		{"git", "git list"},
		{"repo", "repo list"},
		{"hook", "hook list"},
	} {
		t.Run(tc.object, func(t *testing.T) {
			cmd := exec.Command(bin, tc.object, "status")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "ELEGANT_GIT_STATE_FILE="+state)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("run: %v stderr=%s", err, stderr.String())
			}
			surface := tc.object + " status"
			if !strings.Contains(stderr.String(), "deprecated") || !strings.Contains(stderr.String(), surface) {
				t.Fatalf("expected %s deprecation warning, stderr=%q", surface, stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.replacement) {
				t.Fatalf("expected %s replacement hint, stderr=%q", tc.replacement, stderr.String())
			}
		})
	}
}
