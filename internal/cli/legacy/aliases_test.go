package legacy

import (
	"testing"
)

func TestLegacyToPathCoversTwentyCommands(t *testing.T) {
	if len(LegacyToPath) < 19 {
		t.Fatalf("expected at least 19 legacy paths, got %d", len(LegacyToPath))
	}
	if path, ok := LegacyToPath["start-work"]; !ok || path[0] != "work" || path[1] != "start" {
		t.Fatalf("start-work -> %v", path)
	}
}

func TestAliasValueUsesNewForm(t *testing.T) {
	got := AliasValue("start-work")
	want := "elegant work start"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestIDFromLegacy(t *testing.T) {
	id, ok := IDFromLegacy("obtain-work")
	if !ok || id.Action != "track" {
		t.Fatalf("obtain-work -> %+v", id)
	}
	id, ok = IDFromLegacy("show-work")
	if !ok || id.Action != "list" {
		t.Fatalf("show-work -> %+v", id)
	}
}
