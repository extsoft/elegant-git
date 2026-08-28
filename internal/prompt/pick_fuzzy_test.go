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

func TestVisibleChoicesCapsAtTen(t *testing.T) {
	choices := make([]Choice, 15)
	for i := range choices {
		choices[i] = Choice{Value: fmt.Sprintf("o%d", i)}
	}
	visible, sel := visibleChoices(choices, 0)
	if len(visible) != pickMaxVisible || pickMaxVisible != 10 {
		t.Fatalf("visible=%d pickMaxVisible=%d", len(visible), pickMaxVisible)
	}
	if sel != 0 || visible[0].Value != "o0" {
		t.Fatalf("sel=%d first=%q", sel, visible[0].Value)
	}
	visible, sel = visibleChoices(choices, 14)
	if len(visible) != 10 || sel != 9 || visible[9].Value != "o14" {
		t.Fatalf("end window: len=%d sel=%d last=%q", len(visible), sel, visible[len(visible)-1].Value)
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

func TestPickTwoChoicesFilters(t *testing.T) {
	in := strings.NewReader("o\n")
	out := &bytes.Buffer{}
	p := NewTTY(in, out)
	got, err := p.Pick("Branch", []Choice{
		{Value: "main"},
		{Value: "origin/feature"},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "origin/feature" {
		t.Fatalf("got %q", got)
	}
}

func TestPickFallbackThreeChoices(t *testing.T) {
	in := strings.NewReader("feat\n")
	out := &bytes.Buffer{}
	p := NewTTY(in, out)
	got, err := p.Pick("Branch", []Choice{
		{Value: "main"},
		{Value: "origin/feature"},
		{Value: "develop"},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "origin/feature" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatPickLineColumns(t *testing.T) {
	line := formatPickLine(true, false, Choice{Value: "start", Description: "Creates a new branch."}, 5)
	if line != "> start  Creates a new branch." {
		t.Fatalf("got %q", line)
	}
	long := strings.Repeat("x", 80)
	line = formatPickLine(false, false, Choice{Value: "a", Description: truncateDesc(long, descMaxLen)}, 1)
	if !strings.HasPrefix(line, "  a  ") {
		t.Fatalf("got %q", line)
	}
	desc := strings.TrimPrefix(line, "  a  ")
	if len(desc) != descMaxLen {
		t.Fatalf("desc len=%d want %d (%q)", len(desc), descMaxLen, desc)
	}
	if got := formatPickLine(false, true, Choice{Value: "main"}, 4); got != "* main" {
		t.Fatalf("selected mark = %q", got)
	}
	if got := formatPickLine(true, true, Choice{Value: "main"}, 4); got != ">*main" {
		t.Fatalf("both marks = %q", got)
	}
}

func TestPickStatusLineMulti(t *testing.T) {
	got := pickStatusLine(pickScreenInput{Filtered: 2, Total: 5, Multi: true, Selected: 1})
	if got != "2 of 5 (1 selected) ────────────────────" {
		t.Fatalf("got %q", got)
	}
}

func TestTruncateDesc(t *testing.T) {
	if got := truncateDesc("short", 70); got != "short" {
		t.Fatalf("got %q", got)
	}
	got := truncateDesc(strings.Repeat("a", 80), 70)
	if len(got) != 70 || !strings.HasSuffix(got, "...") {
		t.Fatalf("got %q len=%d", got, len(got))
	}
}

func TestBuildPickScreenFilterFirst(t *testing.T) {
	lines, col := buildPickScreen(pickScreenInput{
		Label: "Branch", Filter: "feat",
		Visible: []Choice{{Value: "origin/feature", Description: "upstream main"}},
		Sel:     0, ValueWidth: len("origin/feature"),
		Filtered: 1, Total: 3,
	})
	if len(lines) != 3 {
		t.Fatalf("got %d lines: %v", len(lines), lines)
	}
	if lines[0] != "Branch: feat" {
		t.Fatalf("prompt line = %q", lines[0])
	}
	if col != len("Branch: feat")+1 {
		t.Fatalf("cursorCol=%d", col)
	}
	if lines[1] != "1 of 3 ────────────────────" {
		t.Fatalf("status = %q", lines[1])
	}
	want := "> origin/feature  upstream main"
	if lines[2] != want {
		t.Fatalf("option line = %q want %q", lines[2], want)
	}
}

func TestBuildPickScreenPlaceholder(t *testing.T) {
	lines, col := buildPickScreen(pickScreenInput{
		Label: "What now", Filter: "",
		Visible: nil, Sel: 0, ValueWidth: 0,
		Filtered: 0, Total: 4,
	})
	if lines[0] != "What now: " {
		t.Fatalf("prompt storage = %q", lines[0])
	}
	if col != len("What now: ")+1 {
		t.Fatalf("cursorCol=%d", col)
	}
	if !strings.HasPrefix(lines[1], "0 of 4 ") {
		t.Fatalf("status = %q", lines[1])
	}
	if lines[2] != "  (no matches)" {
		t.Fatalf("empty = %q", lines[2])
	}
}

func TestPaintPickPromptPlaceholder(t *testing.T) {
	got := paintPickPrompt("What now: ", "What now", "")
	if !strings.Contains(got, "\033[3m"+pickPlaceholderSingle+"\033[m") {
		t.Fatalf("expected italic placeholder: %q", got)
	}
	got = paintPickPrompt("What now: list", "What now", "list")
	if strings.Contains(got, pickPlaceholderSingle) {
		t.Fatalf("placeholder should hide when typing: %q", got)
	}
}

func TestWritePickScreenRedrawsInPlace(t *testing.T) {
	out := &bytes.Buffer{}
	first := []string{"What now: ", "4 of 4 ────────────────────", "  main  default", "  feat  origin/feat"}
	writePickScreen(out, first, 0, "What now", "", len("What now: ")+1)
	got := out.String()
	if strings.Count(got, "\n") != 4 {
		t.Fatalf("first draw newlines=%d want 4 (%q)", strings.Count(got, "\n"), got)
	}
	if !strings.Contains(got, "\r\n") {
		t.Fatalf("raw-mode redraw must use CR+LF: %q", got)
	}
	if !strings.Contains(got, "\033[1;34mWhat now: \033[m") {
		t.Fatalf("expected bold blue prompt: %q", got)
	}
	if !strings.Contains(got, "\033[3m"+pickPlaceholderSingle+"\033[m") {
		t.Fatalf("expected italic placeholder: %q", got)
	}
	if !strings.Contains(got, "\033[J") {
		t.Fatalf("expected erase-down before redraw: %q", got)
	}
	if !strings.Contains(got, "\033[4A") {
		t.Fatalf("expected cursor back on prompt: %q", got)
	}
	if !strings.Contains(got, fmt.Sprintf("\033[%dG", len("What now: ")+1)) {
		t.Fatalf("expected cursor at end of prompt: %q", got)
	}
	second := []string{"What now: f", "1 of 4 ────────────────────", "> feat  origin/feat"}
	writePickScreen(out, second, 4, "What now", "f", len("What now: f")+1)
	got = out.String()
	if strings.Count(got, "What now:") != 2 {
		t.Fatalf("expected prompt rewritten once, got %q", got)
	}
	// Second frame paints filter "f" without the italic placeholder.
	if strings.Count(got, pickPlaceholderSingle) != 1 {
		t.Fatalf("placeholder once on empty filter only, got %q", got)
	}
}

func TestFinalizePickScreen(t *testing.T) {
	out := &bytes.Buffer{}
	fmt.Fprintln(out, "prior output")
	finalizePickScreen(out, "What now", "quit", 0)
	got := out.String()
	if !strings.Contains(got, "\033[1;34mWhat now: \033[mquit\r\n\r\n") {
		t.Fatalf("got %q", got)
	}
}

func TestFinalizePickScreenClearsListThenCRLF(t *testing.T) {
	out := &bytes.Buffer{}
	finalizePickScreen(out, "What now", "list", 5)
	got := out.String()
	if !strings.Contains(got, "\033[4A") {
		t.Fatalf("expected cursor back to prompt after erasing 5 lines: %q", got)
	}
	if !strings.Contains(got, "\033[1;34mWhat now: \033[mlist\r\n\r\n") {
		t.Fatalf("got %q", got)
	}
}

func TestIndexOfChoice(t *testing.T) {
	choices := []Choice{{Value: "a"}, {Value: "quit"}, {Value: "b"}}
	if got := indexOfChoice(choices, "quit"); got != 1 {
		t.Fatalf("got %d", got)
	}
	if got := indexOfChoice(choices, ""); got != 0 {
		t.Fatalf("got %d", got)
	}
}
