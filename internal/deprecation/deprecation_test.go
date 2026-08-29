package deprecation

import (
	"bytes"
	"strings"
	"testing"
)

func TestRecordOncePerID(t *testing.T) {
	Reset()
	var buf bytes.Buffer
	SetOutputWriter(&buf)
	Record(DEP002, "personal hook: .workflows/foo", "new/path", "eg repo migrate")
	Record(DEP002, "duplicate", "ignored", "ignored")
	Flush()
	if strings.Count(buf.String(), "warning:") != 1 {
		t.Fatalf("expected 1 warning, got:\n%s", buf.String())
	}
}

func TestRecordLegacyCommand(t *testing.T) {
	Reset()
	var buf bytes.Buffer
	SetOutputWriter(&buf)
	RecordLegacyCommand("start-work", "work start")
	events := Events()
	if len(events) != 1 || events[0].ID != DEP001 {
		t.Fatalf("events = %+v", events)
	}
	if events[0].Replacement != "work start" {
		t.Fatalf("replacement = %q", events[0].Replacement)
	}
	Flush()
	want := "Warning: the `start-work` command is deprecated; please use `work start`. Run `eg git migrate` to migrate automatically.\n"
	if buf.String() != want {
		t.Fatalf("got %q want %q", buf.String(), want)
	}
}
