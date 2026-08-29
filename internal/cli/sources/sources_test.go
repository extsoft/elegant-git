package sources

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
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

func TestWorkspacesWithCreateNew(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	choices, err := WorkspacesWithCreateNew(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(choices) != 1 || choices[0].Value != WorkspaceCreateNew {
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
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/main"
	git.Use(m)

	choices, err := RemoteBranches(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(choices) != 1 || choices[0].Value != "origin/main" || choices[0].Description != "" {
		t.Fatalf("got %+v", choices)
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

func TestLocalBranchesIncludeUpstreamDescription(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "feat\torigin/feat\nmain\t"
	git.Use(m)

	choices, err := LocalBranches(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	byValue := map[string]string{}
	for _, c := range choices {
		byValue[c.Value] = c.Description
	}
	if byValue["feat"] != "origin/feat" {
		t.Fatalf("feat desc=%q", byValue["feat"])
	}
	if byValue["main"] != "" {
		t.Fatalf("main desc=%q", byValue["main"])
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
