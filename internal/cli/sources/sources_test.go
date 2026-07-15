package sources

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bees-hive/elegant-git/internal/git"
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

func TestRemoteBranchesFetchesFirst(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["for-each-ref --format=%(refname:short)\t%(objectname:short) refs/remotes"] = "origin/main\tabc"
	git.Use(m)

	if _, err := RemoteBranches(context.Background()); err != nil {
		t.Fatal(err)
	}
	fetchIdx, refIdx := -1, -1
	for i, c := range m.Calls {
		if len(c.Args) == 0 {
			continue
		}
		switch c.Args[0] {
		case "fetch":
			fetchIdx = i
		case "for-each-ref":
			refIdx = i
		}
	}
	if fetchIdx < 0 {
		t.Fatalf("missing fetch, calls=%v", m.Calls)
	}
	if refIdx < 0 {
		t.Fatalf("missing for-each-ref, calls=%v", m.Calls)
	}
	if fetchIdx > refIdx {
		t.Fatalf("fetch=%d after for-each-ref=%d", fetchIdx, refIdx)
	}
}

func TestRemoteBranchesSkipsFetchWithoutRemotes(t *testing.T) {
	m := git.NewMemoryRunner()
	git.Use(m)

	if _, err := RemoteBranches(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, c := range m.Calls {
		if len(c.Args) > 0 && c.Args[0] == "fetch" {
			t.Fatalf("unexpected fetch: %v", c.Args)
		}
	}
}
