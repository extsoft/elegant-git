package prompt

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		q, target string
		want      bool
	}{
		{"", "anything", true},
		{"feat", "origin/feature", true},
		{"of", "origin/feature", true},
		{"xyz", "origin/feature", false},
		{"Main", "main", true},
	}
	for _, tc := range tests {
		if got := fuzzyMatch(tc.q, tc.target); got != tc.want {
			t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tc.q, tc.target, got, tc.want)
		}
	}
}

func TestFilterChoices(t *testing.T) {
	choices := []Choice{
		{Value: "main"},
		{Value: "origin/feature"},
		{Value: "develop"},
	}
	got := filterChoices(choices, "feat")
	if len(got) != 1 || got[0].Value != "origin/feature" {
		t.Fatalf("got %+v", got)
	}
}

func TestPickSelectedValue(t *testing.T) {
	if got := pickSelectedValue("origin/main\tabc123"); got != "origin/main" {
		t.Fatalf("got %q", got)
	}
	if got := pickSelectedValue("origin/main"); got != "origin/main" {
		t.Fatalf("got %q", got)
	}
}

func TestPickFallback(t *testing.T) {
	in := strings.NewReader("feat\n1\n")
	out := &bytes.Buffer{}
	p := NewTTY(in, out)
	got, err := p.Pick("Branch", []Choice{
		{Value: "main"},
		{Value: "origin/feature"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "origin/feature" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildPickScreenFilterFirst(t *testing.T) {
	lines := buildPickScreen("Branch", "feat", []Choice{
		{Value: "origin/feature", Description: "upstream main"},
	}, 0)
	if len(lines) != 2 {
		t.Fatalf("got %d lines: %v", len(lines), lines)
	}
	if lines[0] != "Branch> feat" {
		t.Fatalf("prompt line = %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "> origin/feature") {
		t.Fatalf("option line = %q", lines[1])
	}
}

func TestFinalizePickScreen(t *testing.T) {
	out := &bytes.Buffer{}
	fmt.Fprintln(out, "prior output")
	finalizePickScreen(out, "Branch", "main", 0)
	if !strings.Contains(out.String(), "Branch> main") {
		t.Fatalf("got %q", out.String())
	}
}
