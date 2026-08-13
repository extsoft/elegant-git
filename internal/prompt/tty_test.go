package prompt

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/bees-hive/elegant-git/internal/text"
)

func captureQuestions(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	text.SetOutput(buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	return buf
}

func TestEditOrAcceptKeepsDefault(t *testing.T) {
	in := strings.NewReader("\n")
	out := &bytes.Buffer{}
	p := NewTTY(in, out)
	got, err := p.EditOrAccept("Editor", "vim")
	if err != nil {
		t.Fatal(err)
	}
	if got != "vim" {
		t.Fatalf("got %q", got)
	}
}

func TestEditOrAcceptReplaces(t *testing.T) {
	in := strings.NewReader("emacs\n")
	out := &bytes.Buffer{}
	p := NewTTY(in, out)
	got, err := p.EditOrAccept("Editor", "vim")
	if err != nil {
		t.Fatal(err)
	}
	if got != "emacs" {
		t.Fatalf("got %q", got)
	}
}

func TestEditOrAcceptLineShape(t *testing.T) {
	q := captureQuestions(t)
	p := NewTTY(strings.NewReader("\n"), &bytes.Buffer{})
	if _, err := p.EditOrAccept("Git user.name", "Alice"); err != nil {
		t.Fatal(err)
	}
	want := RequiredLine("Git user.name", "Alice") + " "
	if q.String() != want {
		t.Fatalf("got %q want %q", q.String(), want)
	}
}

func TestSkippableAutoSkip(t *testing.T) {
	p := NewTTY(strings.NewReader(""), &bytes.Buffer{})
	got, err := Skippable(p, "Editor", "vim", "vim")
	if err != nil {
		t.Fatal(err)
	}
	if got != "vim" {
		t.Fatalf("got %q", got)
	}
}

func TestNonInteractiveEditOrAccept(t *testing.T) {
	p := NewNonInteractive()
	got, err := p.EditOrAccept("k", "suggested")
	if err != nil || got != "suggested" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestNonInteractiveBatchChoiceSkip(t *testing.T) {
	p := NewNonInteractive()
	d, err := p.BatchChoice("q", "yes")
	if err != nil || d != BatchSkip {
		t.Fatalf("got %v err=%v", d, err)
	}
}

func TestStringAcceptsDefault(t *testing.T) {
	q := captureQuestions(t)
	p := NewTTY(strings.NewReader("\n"), &bytes.Buffer{})
	got, err := p.String("Git user.name", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Alice" {
		t.Fatalf("got %q", got)
	}
	want := RequiredLine("Git user.name", "Alice") + " "
	if q.String() != want {
		t.Fatalf("got %q want %q", q.String(), want)
	}
}

func TestStringRequiredReasks(t *testing.T) {
	q := captureQuestions(t)
	p := NewTTY(strings.NewReader("\nsome name\n"), &bytes.Buffer{})
	got, err := p.String("Profile name", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "some name" {
		t.Fatalf("got %q", got)
	}
	line := RequiredLine("Profile name", "") + " "
	if q.String() != line+line {
		t.Fatalf("got %q want two prompts", q.String())
	}
}

func TestConfirmDefaultYes(t *testing.T) {
	p := NewTTY(strings.NewReader("\n"), &bytes.Buffer{})
	ok, err := p.Confirm("Proceed?", true)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected true")
	}
}

func TestConfirmDefaultNo(t *testing.T) {
	q := captureQuestions(t)
	p := NewTTY(strings.NewReader("\n"), &bytes.Buffer{})
	ok, err := p.Confirm("Proceed?", false)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false")
	}
	want := ClosedLine("Proceed?", yesNoOptions, "no") + " "
	if q.String() != want {
		t.Fatalf("got %q want %q", q.String(), want)
	}
}

func TestConfirmWordAndLetter(t *testing.T) {
	tests := []struct {
		in   string
		def  bool
		want bool
	}{
		{"yes\n", false, true},
		{"y\n", false, true},
		{"Y\n", false, true},
		{"no\n", true, false},
		{"n\n", true, false},
	}
	for _, tc := range tests {
		p := NewTTY(strings.NewReader(tc.in), &bytes.Buffer{})
		got, err := p.Confirm("Proceed?", tc.def)
		if err != nil {
			t.Fatalf("input %q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("input %q: got %v want %v", tc.in, got, tc.want)
		}
	}
}

func TestConfirmInvalidReasks(t *testing.T) {
	q := captureQuestions(t)
	p := NewTTY(strings.NewReader("x\ny\n"), &bytes.Buffer{})
	ok, err := p.Confirm("Proceed?", false)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected true")
	}
	line := ClosedLine("Proceed?", yesNoOptions, "no") + " "
	if q.String() != line+line {
		t.Fatalf("got %q want two prompts", q.String())
	}
}

func TestBatchChoiceDefault(t *testing.T) {
	p := NewTTY(strings.NewReader("\n"), &bytes.Buffer{})
	d, err := p.BatchChoice("Apply to repo?", "no")
	if err != nil {
		t.Fatal(err)
	}
	if d != BatchReject {
		t.Fatalf("got %v", d)
	}
}

func TestBatchChoiceWordsAndLetters(t *testing.T) {
	tests := []struct {
		in   string
		want BatchDecision
	}{
		{"yes\n", BatchConfirm},
		{"y\n", BatchConfirm},
		{"no\n", BatchReject},
		{"n\n", BatchReject},
		{"all\n", BatchApplyAll},
		{"a\n", BatchApplyAll},
		{"skip\n", BatchSkip},
		{"s\n", BatchSkip},
	}
	for _, tc := range tests {
		p := NewTTY(strings.NewReader(tc.in), &bytes.Buffer{})
		got, err := p.BatchChoice("Apply to repo?", "no")
		if err != nil {
			t.Fatalf("input %q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("input %q: got %v want %v", tc.in, got, tc.want)
		}
	}
}

func TestBatchChoiceEmptyNoDefaultReasks(t *testing.T) {
	q := captureQuestions(t)
	p := NewTTY(strings.NewReader("\nskip\n"), &bytes.Buffer{})
	d, err := p.BatchChoice("Apply to repo?", "")
	if err != nil {
		t.Fatal(err)
	}
	if d != BatchSkip {
		t.Fatalf("got %v", d)
	}
	line := ClosedLine("Apply to repo?", batchChoiceOptions, "") + " "
	if q.String() != line+line {
		t.Fatalf("got %q want two prompts", q.String())
	}
}

func TestOptionalLeavesUnset(t *testing.T) {
	q := captureQuestions(t)
	p := NewTTY(strings.NewReader("\n"), &bytes.Buffer{})
	got, err := p.Optional("Signing key", "ABC123")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("got %q", got)
	}
	want := OptionalLine("Signing key", "ABC123") + " "
	if q.String() != want {
		t.Fatalf("got %q want %q", q.String(), want)
	}
}

func TestClosedOptionalEmpty(t *testing.T) {
	p := NewTTY(strings.NewReader("\n"), &bytes.Buffer{})
	got, err := p.Closed("Hook location", []string{"personal", "common"}, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestPickFallbackFilterThenClosed(t *testing.T) {
	q := captureQuestions(t)
	p := NewTTY(strings.NewReader("m\nmain\n"), &bytes.Buffer{})
	got, err := p.Pick("Start from ref", []Choice{{Value: "main"}, {Value: "master"}, {Value: "develop"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "main" {
		t.Fatalf("got %q", got)
	}
	filter := RequiredLine("Start from ref", "") + " "
	closed := ClosedLine("Start from ref", []string{"main", "master"}, "") + " "
	if q.String() != filter+closed {
		t.Fatalf("got %q want filter then closed", q.String())
	}
}

func TestPickFallbackSingleMatch(t *testing.T) {
	p := NewTTY(strings.NewReader("dev\n"), &bytes.Buffer{})
	got, err := p.Pick("Start from ref", []Choice{{Value: "main"}, {Value: "develop"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "develop" {
		t.Fatalf("got %q", got)
	}
}
