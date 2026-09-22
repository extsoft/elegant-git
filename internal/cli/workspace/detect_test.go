package workspace

import (
	"slices"
	"testing"

	"github.com/extsoft/elegant-git/internal/cli/catalog"
)

func TestDetectAlwaysAsks(t *testing.T) {
	for _, tc := range []struct {
		name  string
		snap  snapshot
		steps []string
		opts  []string
	}{
		{
			name:  "outside git with workspaces",
			snap:  snapshot{WorkspaceCount: 2},
			steps: []string{"in a git repository? no", "selected: ask"},
			opts:  []string{"list", "new", "edit", "delete", "doctor", "help", "quit"},
		},
		{
			name:  "outside git no workspaces",
			snap:  snapshot{},
			steps: []string{"in a git repository? no", "selected: ask"},
			opts:  []string{"new", "help", "quit"},
		},
		{
			name:  "unlinked with workspaces",
			snap:  snapshot{InGit: true, WorkspaceCount: 1},
			steps: []string{"in a git repository? yes", "workspace linked? no", "selected: ask"},
			opts:  []string{"new", "link", "doctor", "help", "quit"},
		},
		{
			name:  "unlinked no workspaces",
			snap:  snapshot{InGit: true},
			steps: []string{"in a git repository? yes", "workspace linked? no", "selected: ask"},
			opts:  []string{"new", "help", "quit"},
		},
		{
			name:  "linked",
			snap:  snapshot{InGit: true, Linked: true, WorkspaceName: "github", WorkspaceCount: 1},
			steps: []string{"in a git repository? yes", "workspace linked? yes (github)", "selected: ask"},
			opts:  []string{"list", "new", "link", "edit", "delete", "fetch", "doctor", "help", "quit"},
		},
		{
			name:  "linked no other workspaces count edge",
			snap:  snapshot{InGit: true, Linked: true, WorkspaceName: "solo", WorkspaceCount: 0},
			steps: []string{"in a git repository? yes", "workspace linked? yes (solo)", "selected: ask"},
			opts:  []string{"new", "list", "fetch", "help", "quit"},
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
				if catalog.Purpose("workspace", c.Value) == "" || c.Description != catalog.Purpose("workspace", c.Value) {
					t.Fatalf("missing description for %q", c.Value)
				}
				wantDisplay := ""
				if c.Value != "quit" {
					wantDisplay = "workspace " + c.Value
				}
				if c.Display != wantDisplay {
					t.Fatalf("display for %q=%q want %q", c.Value, c.Display, wantDisplay)
				}
			}
		})
	}
}
