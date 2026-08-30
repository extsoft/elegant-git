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
	want := "!eg work start"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestShowWorkflowsMapsToHookList(t *testing.T) {
	path, ok := LegacyToPath["show-workflows"]
	if !ok || len(path) != 2 || path[0] != "hook" || path[1] != "list" {
		t.Fatalf("show-workflows -> %v", path)
	}
	id, ok := LegacyToID["show-workflows"]
	if !ok || id.Action != "list" {
		t.Fatalf("show-workflows id -> %+v", id)
	}
	if got := AliasValue("show-workflows"); got != "!eg hook list" {
		t.Fatalf("got %q", got)
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
