package memory

import (
	"slices"
	"testing"
)

func TestNewCommandSubcommands(t *testing.T) {
	c := NewCommand()
	var uses []string
	for _, sub := range c.Commands() {
		if sub.Hidden {
			continue
		}
		uses = append(uses, sub.Name())
	}
	want := []string{"list", "workspaces", "repositories"}
	slices.Sort(uses)
	slices.Sort(want)
	if !slices.Equal(uses, want) {
		t.Fatalf("got %v want %v", uses, want)
	}
}
