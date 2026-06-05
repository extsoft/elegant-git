package sources

import (
	"context"
	"path/filepath"
	"testing"
)

func TestHookCommandIDsFromProvider(t *testing.T) {
	SetHookCommandIDsProvider(func() []string { return []string{"work.start", "repo.clone"} })
	choices, err := HookCommandIDs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(choices) != 2 || choices[0].Value != "work.start" {
		t.Fatalf("got %+v", choices)
	}
}

func TestProfilesWithCreateNew(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	choices, err := ProfilesWithCreateNew(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(choices) != 1 || choices[0].Value != ProfileCreateNew {
		t.Fatalf("got %+v", choices)
	}
}

func TestHookTypes(t *testing.T) {
	choices, err := HookTypes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(choices) != 2 {
		t.Fatalf("got %d", len(choices))
	}
}

func TestCompletionShells(t *testing.T) {
	choices, err := CompletionShells(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(choices) != 4 {
		t.Fatalf("got %d", len(choices))
	}
}
