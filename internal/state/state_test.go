package state

import (
	"fmt"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
)

func TestUpstreamOfEmptyWhenUnset(t *testing.T) {
	mem := git.NewMemoryRunner()
	mem.FailOn["rev-parse --abbrev-ref 1635@{upstream}"] = fmt.Errorf("fatal: no upstream configured for branch '1635'")
	git.Use(mem)
	got := UpstreamOf("1635")
	if got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestUpstreamOfReturnsTrackingBranch(t *testing.T) {
	mem := git.NewMemoryRunner()
	mem.Outputs["rev-parse --abbrev-ref feature@{upstream}"] = "origin/feature\n"
	git.Use(mem)
	got := UpstreamOf("feature")
	if got != "origin/feature" {
		t.Fatalf("got %q", got)
	}
}
