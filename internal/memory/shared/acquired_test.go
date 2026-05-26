package shared

import (
	"path/filepath"
	"testing"
)

func TestSetAcquiredRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	s := emptyState()
	SetAcquired(s, "1.2.3")
	if err := Save(s); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if Acquired(loaded) != "1.2.3" {
		t.Fatalf("got %q", Acquired(loaded))
	}
}
