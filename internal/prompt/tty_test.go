package prompt

import (
	"bytes"
	"strings"
	"testing"
)

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
	d, err := p.BatchChoice("q")
	if err != nil || d != BatchSkip {
		t.Fatalf("got %v err=%v", d, err)
	}
}
