package work

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/extsoft/elegant-git/internal/cli/catalog"
	"github.com/extsoft/elegant-git/internal/git"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name    string
		snap    snapshot
		action  string
		thenAsk bool
		steps   []string
	}{
		{
			name:   "rebase accept helper",
			snap:   snapshot{Rebasing: true, RebasingBranch: acceptWorkBranch},
			action: "accept",
			steps: []string{
				"rebase in progress? yes (" + acceptWorkBranch + ")",
				"selected: eg work accept",
			},
		},
		{
			name:   "rebase feature",
			snap:   snapshot{Rebasing: true, RebasingBranch: "feat"},
			action: "polish",
			steps: []string{
				"rebase in progress? yes (feat)",
				"selected: eg work polish",
			},
		},
		{
			name:   "dirty protected",
			snap:   snapshot{Branch: "main", Protected: true, Dirty: true},
			action: "start",
			steps: []string{
				"rebase in progress? no",
				"on protected branch 'main'? yes",
				"uncommitted changes? yes",
				"selected: eg work start",
			},
		},
		{
			name:   "dirty feature",
			snap:   snapshot{Branch: "feat", Dirty: true},
			action: "save",
			steps: []string{
				"rebase in progress? no",
				"on protected branch 'feat'? no",
				"uncommitted changes? yes",
				"selected: eg work save",
			},
		},
		{
			name:    "behind only",
			snap:    snapshot{Branch: "feat", HasUpstream: true, Ahead: 0, Behind: 3},
			action:  "sync",
			thenAsk: true,
			steps: []string{
				"rebase in progress? no",
				"on protected branch 'feat'? no",
				"uncommitted changes? no",
				"behind upstream only? yes (ahead 0, behind 3)",
				"selected: eg work sync",
			},
		},
		{
			name:    "idle list",
			snap:    snapshot{Branch: "main", Protected: true, HasUpstream: true, Ahead: 0, Behind: 0},
			action:  "list",
			thenAsk: true,
			steps: []string{
				"rebase in progress? no",
				"on protected branch 'main'? yes",
				"uncommitted changes? no",
				"behind upstream only? no (ahead 0, behind 0)",
				"unique commits vs source? no",
				"selected: eg work list",
			},
		},
		{
			name:    "unique commits ask",
			snap:    snapshot{Branch: "feat", HasUpstream: true, Ahead: 2, Behind: 0, UniqueCommits: true},
			thenAsk: true,
			steps: []string{
				"rebase in progress? no",
				"on protected branch 'feat'? no",
				"uncommitted changes? no",
				"behind upstream only? no (ahead 2, behind 0)",
				"unique commits vs source? yes",
				"selected: ask",
			},
		},
		{
			name:    "detached dirty ask",
			snap:    snapshot{Branch: "HEAD", Detached: true, Dirty: true, Remotes: true},
			thenAsk: true,
			steps: []string{
				"rebase in progress? no",
				"on protected branch 'HEAD'? no",
				"uncommitted changes? yes",
				"selected: ask",
			},
		},
		{
			name:    "no upstream idle list",
			snap:    snapshot{Branch: "feat"},
			action:  "list",
			thenAsk: true,
			steps: []string{
				"rebase in progress? no",
				"on protected branch 'feat'? no",
				"uncommitted changes? no",
				"behind upstream only? no (no upstream)",
				"unique commits vs source? no",
				"selected: eg work list",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := detect(tc.snap)
			if got.Action != tc.action || got.ThenAsk != tc.thenAsk {
				t.Fatalf("action=%q thenAsk=%v; want %q %v", got.Action, got.ThenAsk, tc.action, tc.thenAsk)
			}
			if !slices.Equal(got.Steps, tc.steps) {
				t.Fatalf("steps=%q\nwant  %q", got.Steps, tc.steps)
			}
		})
	}
}

func TestAskOptions(t *testing.T) {
	tests := []struct {
		name string
		snap snapshot
		want []string
	}{
		{
			name: "protected clean",
			snap: snapshot{Protected: true},
			want: []string{"start", "track", "accept", "list", "help", "quit"},
		},
		{
			name: "unique ahead",
			snap: snapshot{UniqueCommits: true, HasUpstream: true, Ahead: 2},
			want: []string{"polish", "push", "accept", "list", "help", "quit"},
		},
		{
			name: "unique no upstream",
			snap: snapshot{UniqueCommits: true},
			want: []string{"polish", "push", "accept", "list", "help", "quit"},
		},
		{
			name: "diverged",
			snap: snapshot{HasUpstream: true, Ahead: 1, Behind: 2, UniqueCommits: true},
			want: []string{"sync", "push", "accept", "list", "help", "quit"},
		},
		{
			name: "detached remotes",
			snap: snapshot{Detached: true, Remotes: true},
			want: []string{"start", "track", "help", "quit"},
		},
		{
			name: "detached no remotes",
			snap: snapshot{Detached: true},
			want: []string{"start", "help", "quit"},
		},
		{
			name: "idle feature remotes",
			snap: snapshot{Remotes: true},
			want: []string{"start", "accept", "list", "push", "track", "help", "quit"},
		},
		{
			name: "idle feature no remotes",
			snap: snapshot{},
			want: []string{"start", "accept", "list", "help", "quit"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := askOptions(tc.snap)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
			choices := askChoices(tc.snap)
			if len(choices) != len(tc.want) {
				t.Fatalf("choices len=%d want %d", len(choices), len(tc.want))
			}
			for i, c := range choices {
				if c.Value != tc.want[i] {
					t.Fatalf("choice[%d]=%q want %q", i, c.Value, tc.want[i])
				}
				if catalog.Purpose("work", c.Value) == "" || c.Description != catalog.Purpose("work", c.Value) {
					t.Fatalf("choice[%d] desc=%q", i, c.Description)
				}
				wantDisplay := ""
				if c.Value != "quit" {
					wantDisplay = "work " + c.Value
				}
				if c.Display != wantDisplay {
					t.Fatalf("choice[%d] display=%q want %q", i, c.Display, wantDisplay)
				}
			}
		})
	}
}

func TestRelevantActionsDirtySaveFirst(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(dir, "repo-state.json"))
	m := git.NewMemoryRunner()
	m.Repo.CurrentBranch = "feat"
	m.Outputs["rev-parse --git-dir"] = filepath.Join(dir, ".git")
	m.FailOn["diff-index --quiet HEAD"] = fmt.Errorf("dirty")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	got := relevantFrom(inspect())
	if len(got) == 0 || got[0] != "save" {
		t.Fatalf("got %v", got)
	}
	if slices.Contains(got, "help") || slices.Contains(got, "quit") || slices.Contains(got, "sync") {
		t.Fatalf("got %v", got)
	}
}

func TestRelevantActionsSyncWhenBehind(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(dir, "repo-state.json"))
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(dir, ".git")
	m.Outputs["rev-list --left-right --count HEAD...@{upstream}"] = "0\t3"
	m.FailOn["diff-index --quiet HEAD"] = fmt.Errorf("dirty")
	m.Repo.CurrentBranch = "feat"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	got := relevantFrom(inspect())
	if len(got) < 2 || got[0] != "sync" || got[1] != "save" {
		t.Fatalf("dirty behind: %v", got)
	}

	m.FailOn["diff-index --quiet HEAD"] = nil
	m.Outputs["rev-list --left-right --count HEAD...@{upstream}"] = "1\t2"
	m.Repo.CurrentBranch = "main"
	got = relevantFrom(inspect())
	if len(got) == 0 || got[0] != "sync" {
		t.Fatalf("protected diverged: %v", got)
	}

	m.Repo.CurrentBranch = "HEAD"
	got = relevantFrom(inspect())
	if slices.Contains(got, "sync") {
		t.Fatalf("detached: %v", got)
	}
}

func TestAcceptArgs(t *testing.T) {
	if !slices.Equal(acceptArgs(snapshot{Branch: "feat"}), []string{"feat"}) {
		t.Fatal("feat")
	}
	if args := acceptArgs(snapshot{Branch: "main", Protected: true}); len(args) != 0 {
		t.Fatalf("protected %v", args)
	}
	if args := acceptArgs(snapshot{Branch: "HEAD", Detached: true}); len(args) != 0 {
		t.Fatalf("detached %v", args)
	}
	if args := acceptArgs(snapshot{}); len(args) != 0 {
		t.Fatalf("empty %v", args)
	}
}

func TestAheadBehind(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["rev-list --left-right --count HEAD...@{upstream}"] = "2\t3"
	git.Use(m)
	ahead, behind := aheadBehind()
	if ahead != 2 || behind != 3 {
		t.Fatalf("ahead=%d behind=%d", ahead, behind)
	}
}
