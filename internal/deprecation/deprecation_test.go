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
	Record(DEP002, "personal hook: .workflows/foo", "new/path", "eg repo doctor")
	Record(DEP002, "duplicate", "ignored", "ignored")
	Flush()
	if strings.Count(buf.String(), "warning:") != 1 {
		t.Fatalf("expected 1 warning, got:\n%s", buf.String())
	}
}

func TestRecordRenamed(t *testing.T) {
	Reset()
	var buf bytes.Buffer
	SetOutputWriter(&buf)
	RecordRenamed(DEP017, "workspace status", "workspace list current")
	events := Events()
	if len(events) != 1 || events[0].ID != DEP017 {
		t.Fatalf("events = %+v", events)
	}
	Flush()
	want := "warning: workspace status is deprecated; use workspace list current.\n"
	if buf.String() != want {
		t.Fatalf("got %q want %q", buf.String(), want)
	}

	Reset()
	buf.Reset()
	SetOutputWriter(&buf)
	RecordRenamed(DEP018, "git status", "git list")
	events = Events()
	if len(events) != 1 || events[0].ID != DEP018 {
		t.Fatalf("events = %+v", events)
	}
	Flush()
	want = "warning: git status is deprecated; use git list.\n"
	if buf.String() != want {
		t.Fatalf("got %q want %q", buf.String(), want)
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
	want := "Warning: the `start-work` command is deprecated; please use `work start`.\n"
	if buf.String() != want {
		t.Fatalf("got %q want %q", buf.String(), want)
	}
}
