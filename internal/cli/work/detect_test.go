package work

import (
	"slices"
	"testing"

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
				"selected: git elegant work accept",
			},
		},
		{
			name:   "rebase feature",
			snap:   snapshot{Rebasing: true, RebasingBranch: "feat"},
			action: "polish",
			steps: []string{
				"rebase in progress? yes (feat)",
				"selected: git elegant work polish",
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
				"selected: git elegant work start",
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
				"selected: git elegant work save",
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
				"selected: git elegant work sync",
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
				"selected: git elegant work list",
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
				"selected: git elegant work list",
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
			want: []string{"start", "track", "accept", "list", "quit"},
		},
		{
			name: "unique ahead",
			snap: snapshot{UniqueCommits: true, HasUpstream: true, Ahead: 2},
			want: []string{"polish", "push", "accept", "list", "quit"},
		},
		{
			name: "unique no upstream",
			snap: snapshot{UniqueCommits: true},
			want: []string{"polish", "push", "accept", "list", "quit"},
		},
		{
			name: "diverged",
			snap: snapshot{HasUpstream: true, Ahead: 1, Behind: 2, UniqueCommits: true},
			want: []string{"sync", "push", "accept", "list", "quit"},
		},
		{
			name: "detached remotes",
			snap: snapshot{Detached: true, Remotes: true},
			want: []string{"start", "track", "quit"},
		},
		{
			name: "detached no remotes",
			snap: snapshot{Detached: true},
			want: []string{"start", "quit"},
		},
		{
			name: "idle feature remotes",
			snap: snapshot{Remotes: true},
			want: []string{"start", "accept", "list", "track", "quit"},
		},
		{
			name: "idle feature no remotes",
			snap: snapshot{},
			want: []string{"start", "accept", "list", "quit"},
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
				if workActionPurpose[c.Value] == "" || c.Description != workActionPurpose[c.Value] {
					t.Fatalf("choice[%d] desc=%q", i, c.Description)
				}
			}
		})
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
