package cli

import (
	"slices"
	"testing"
)

func findGroupActions(object string) []string {
	for _, g := range commandGroups {
		if g.object == object {
			var actions []string
			for _, c := range g.commands {
				actions = append(actions, c.action)
			}
			return actions
		}
	}
	return nil
}

func hasAction(actions []string, action string) bool {
	return slices.Contains(actions, action)
}

func TestMemoryGroupActions(t *testing.T) {
	got := findGroupActions("memory")
	want := []string{"list", "workspaces", "repositories"}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestListActionsInGroups(t *testing.T) {
	for _, tc := range []struct {
		object string
		has    string
		lacks  string
	}{
		{"git", "list", "status"},
		{"workspace", "list", "status"},
		{"repo", "list", "status"},
		{"hook", "list", "status"},
		{"memory", "list", "status"},
		{"work", "list", "status"},
	} {
		actions := findGroupActions(tc.object)
		if !hasAction(actions, tc.has) {
			t.Errorf("%s missing %q in %v", tc.object, tc.has, actions)
		}
		if tc.lacks != "" && hasAction(actions, tc.lacks) {
			t.Errorf("%s should not have %q", tc.object, tc.lacks)
		}
	}
}
