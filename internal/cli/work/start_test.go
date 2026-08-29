package work

import (
	"fmt"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
)

var errNoUpstream = fmt.Errorf("no upstream")

func TestResolveStartPoint(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --abbrev-ref main@{upstream}"] = "origin/main"
	m.FailOn["rev-parse --abbrev-ref origin/feat/foo@{upstream}"] = errNoUpstream
	git.Use(m)

	if got := resolveStartPoint("main"); got != "origin/main" {
		t.Fatalf("resolveStartPoint(main) = %q, want origin/main", got)
	}
	if got := resolveStartPoint("origin/feat/foo"); got != "origin/feat/foo" {
		t.Fatalf("resolveStartPoint(remote) = %q, want origin/feat/foo", got)
	}
}

func TestParseStartChangesChoice(t *testing.T) {
	tests := []struct {
		in   string
		mode string
		ok   bool
	}{
		{"", "stash", true},
		{"a", "stash", true},
		{"add", "stash", true},
		{"R", "reset", true},
		{"reset", "reset", true},
		{"c", "cancel", true},
		{"cancel", "cancel", true},
		{"x", "", false},
	}
	for _, tc := range tests {
		mode, ok := parseStartChangesChoice(tc.in)
		if mode != tc.mode || ok != tc.ok {
			t.Errorf("parseStartChangesChoice(%q) = %q, %v; want %q, %v", tc.in, mode, ok, tc.mode, tc.ok)
		}
	}
}
