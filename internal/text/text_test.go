package text

import (
	"bytes"
	"strings"
	"testing"
)

func TestCommandTextNoTTYPlain(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	CommandText("git", "status")
	got := buf.String()
	if !strings.Contains(got, "==>>") {
		t.Fatalf("missing prefix: %q", got)
	}
	if !strings.Contains(got, "git status") {
		t.Fatalf("missing command: %q", got)
	}
}

func TestPlainText(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	PlainText("rebase in progress? no")
	if got := buf.String(); got != "rebase in progress? no\n" {
		t.Fatalf("got %q", got)
	}
}

func TestInfoBoxPlain(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	InfoBox("Configuring basics...")
	if !strings.Contains(buf.String(), "Configuring basics...") {
		t.Fatalf("unexpected: %q", buf.String())
	}
}
