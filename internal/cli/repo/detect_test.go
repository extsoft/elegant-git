package repo

import (
	"slices"
	"testing"

	"github.com/extsoft/elegant-git/internal/cli/catalog"
	"github.com/extsoft/elegant-git/internal/git"
)

func TestInspect(t *testing.T) {
	for _, tc := range []struct {
		name       string
		gitDir     string
		repoID     string
		inGit      bool
		configured bool
	}{
		{name: "outside git"},
		{name: "in git not configured", gitDir: "/repo/.git", inGit: true},
		{name: "configured", gitDir: "/repo/.git", repoID: "abc", inGit: true, configured: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := git.NewMemoryRunner()
			if tc.gitDir != "" {
				m.Outputs["rev-parse --git-dir"] = tc.gitDir
			}
			if tc.repoID != "" {
				m.Repo.LocalConfig["elegant-git.repo-id"] = tc.repoID
			}
			git.Use(m)
			t.Cleanup(func() { git.Use(git.RealRunner{}) })
			s := inspect()
			if s.InGit != tc.inGit || s.Configured != tc.configured {
				t.Fatalf("got %+v", s)
			}
		})
	}
}

func TestDetectAlwaysAsks(t *testing.T) {
	for _, tc := range []struct {
		name  string
		snap  snapshot
		steps []string
		opts  []string
	}{
		{
			name:  "outside git",
			snap:  snapshot{},
			steps: []string{"in a git repository? no", "selected: ask"},
			opts:  []string{"clone", "init", "list", "help", "quit"},
		},
		{
			name:  "in git not configured",
			snap:  snapshot{InGit: true},
			steps: []string{"in a git repository? yes", "configured? no", "selected: ask"},
			opts:  []string{"configure", "list", "prune", "doctor", "help", "quit"},
		},
		{
			name:  "configured",
			snap:  snapshot{InGit: true, Configured: true},
			steps: []string{"in a git repository? yes", "configured? yes", "selected: ask"},
			opts:  []string{"list", "sync", "prune", "configure", "doctor", "help", "quit"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := detect(tc.snap)
			if !slices.Equal(d.Steps, tc.steps) {
				t.Fatalf("steps=%v want %v", d.Steps, tc.steps)
			}
			if !slices.Equal(askOptions(tc.snap), tc.opts) {
				t.Fatalf("opts=%v want %v", askOptions(tc.snap), tc.opts)
			}
			for _, c := range askChoices(tc.snap) {
				if catalog.Purpose("repo", c.Value) == "" || c.Description != catalog.Purpose("repo", c.Value) {
					t.Fatalf("missing description for %q", c.Value)
				}
				wantDisplay := ""
				if c.Value != "quit" {
					wantDisplay = "repo " + c.Value
				}
				if c.Display != wantDisplay {
					t.Fatalf("display for %q=%q want %q", c.Value, c.Display, wantDisplay)
				}
			}
		})
	}
}
