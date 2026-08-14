package work

import (
	"context"
	"testing"

	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/spf13/cobra"
)

func TestAcceptLogicContinuesHelperRebaseWithoutBranch(t *testing.T) {
	m := git.NewMemoryRunner()
	git.Use(m)
	orig := isAcceptHelperRebase
	isAcceptHelperRebase = func() bool { return true }
	defer func() { isAcceptHelperRebase = orig }()

	var branch string
	spec := acceptSpec(&branch)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	if err := acceptLogic(cmd, nil, spec); err != nil {
		t.Fatal(err)
	}
	if len(m.Calls) != 1 || m.Calls[0].Args[0] != "rebase" || m.Calls[0].Args[1] != "--continue" {
		t.Fatalf("calls=%v", m.Calls)
	}
	if branch != "" {
		t.Fatalf("branch=%q", branch)
	}
}
