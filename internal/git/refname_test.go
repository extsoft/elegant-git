package git

import (
	"os/exec"
	"strings"
	"testing"
)

func TestValidateBranchName(t *testing.T) {
	valid := []string{"main", "feature/foo", "datadog-alerts", "OPS-2162"}
	for _, name := range valid {
		if err := validateBranchName(name); err != nil {
			t.Errorf("%q: %v", name, err)
		}
	}
	invalid := []string{"", ".hidden", "bad name", "foo..bar", "ends/", "trail.", "has@{", "a.lock"}
	for _, name := range invalid {
		if err := validateBranchName(name); err == nil {
			t.Errorf("%q: expected error", name)
		}
	}
}

func TestNewBranchNameProblem(t *testing.T) {
	branches := []string{"main", "feature", "a/b"}
	tests := []struct {
		name string
		want string
	}{
		{"main", "branch 'main' already exists"},
		{"feature/x", "branch 'feature/x' conflicts with existing branch 'feature'"},
		{"a", "branch 'a' conflicts with existing branch 'a/b'"},
		{"Main", ""},
		{"feature", "branch 'feature' already exists"},
		{"ok", ""},
		{"", "branch name is required"},
		{"bad name", "branch name cannot contain ' '"},
		{"a/b/c", "branch 'a/b/c' conflicts with existing branch 'a/b'"},
	}
	for _, tc := range tests {
		got := NewBranchNameProblem(tc.name, branches)
		if got != tc.want {
			t.Errorf("NewBranchNameProblem(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
	if got := NewBranchNameProblem("topic", []string{"other", "topic/x"}); got != "branch 'topic' conflicts with existing branch 'topic/x'" {
		t.Fatalf("got %q", got)
	}
	if got := NewBranchNameProblem("bad name", []string{"bad name"}); got != "branch name cannot contain ' '" {
		t.Fatalf("format before exists: %q", got)
	}
}

func TestCheckBranchNameMemoryRunner(t *testing.T) {
	m := NewMemoryRunner()
	Use(m)
	t.Cleanup(func() { Use(RealRunner{}) })
	if err := CheckBranchName("datadog-alerts"); err != nil {
		t.Fatal(err)
	}
	if err := CheckBranchName("bad name"); err == nil {
		t.Fatal("expected error for space in branch name")
	}
}

func TestCheckBranchNameReportsGitText(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	Use(RealRunner{})
	t.Cleanup(func() { Use(RealRunner{}) })
	for _, name := range []string{"HEAD", "-x", "a/.b", "a.lock/b"} {
		err := CheckBranchName(name)
		if err == nil || strings.Contains(err.Error(), "exit status") || !strings.Contains(err.Error(), name) {
			t.Fatalf("%q: %v", name, err)
		}
	}
}
