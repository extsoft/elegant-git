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

func TestColoredBufferFalse(t *testing.T) {
	var buf bytes.Buffer
	if Colored(&buf) {
		t.Fatal("buffer is not a TTY")
	}
}

func TestFinfoNoTTYPlain(t *testing.T) {
	var buf bytes.Buffer
	Finfo(&buf, "Branch")
	if got := buf.String(); got != "Branch\n" {
		t.Fatalf("got %q", got)
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

func TestOrUnset(t *testing.T) {
	if OrUnset("") != "(unset)" {
		t.Fatal("empty")
	}
	if OrUnset("x") != "x" {
		t.Fatal("value")
	}
}
