package workspace

import (
	"slices"
	"testing"
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
			opts:  []string{"list", "new", "edit", "delete", "quit"},
		},
		{
			name:  "outside git no workspaces",
			snap:  snapshot{},
			steps: []string{"in a git repository? no", "selected: ask"},
			opts:  []string{"new", "quit"},
		},
		{
			name:  "unlinked with workspaces",
			snap:  snapshot{InGit: true, WorkspaceCount: 1},
			steps: []string{"in a git repository? yes", "workspace linked? no", "selected: ask"},
			opts:  []string{"new", "link", "quit"},
		},
		{
			name:  "unlinked no workspaces",
			snap:  snapshot{InGit: true},
			steps: []string{"in a git repository? yes", "workspace linked? no", "selected: ask"},
			opts:  []string{"new", "quit"},
		},
		{
			name:  "linked",
			snap:  snapshot{InGit: true, Linked: true, WorkspaceName: "github", WorkspaceCount: 1},
			steps: []string{"in a git repository? yes", "workspace linked? yes (github)", "selected: ask"},
			opts:  []string{"list", "new", "link", "edit", "delete", "status", "fetch", "quit"},
		},
		{
			name:  "linked no other workspaces count edge",
			snap:  snapshot{InGit: true, Linked: true, WorkspaceName: "solo", WorkspaceCount: 0},
			steps: []string{"in a git repository? yes", "workspace linked? yes (solo)", "selected: ask"},
			opts:  []string{"new", "status", "fetch", "quit"},
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
				if c.Description == "" {
					t.Fatalf("missing description for %q", c.Value)
				}
			}
		})
	}
}
