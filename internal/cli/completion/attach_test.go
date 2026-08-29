package completion

import (
	"context"
	"testing"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/spf13/cobra"
)

func TestAttachValidArgsFunction(t *testing.T) {
	var val string
	complete := func(context.Context) ([]argspec.Choice, error) {
		return []argspec.Choice{{Value: "bash", Description: "Bourne Again"}}, nil
	}
	spec := argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInputWithComplete("shell", 0, true, "Shell", &val, nil, complete, true),
	}}
	cmd := &cobra.Command{Use: "test"}
	Attach(cmd, spec)
	if cmd.ValidArgsFunction == nil {
		t.Fatal("expected ValidArgsFunction")
	}
	out, dir := cmd.ValidArgsFunction(cmd, nil, "")
	if len(out) != 1 || out[0] != "bash\tBourne Again" {
		t.Fatalf("got %v", out)
	}
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("dir=%v", dir)
	}
}

func TestAttachArgsLimitsPositionals(t *testing.T) {
	spec := argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInput("a", 0, true, "first", new(string), nil),
		argspec.PositionalInput("b", 1, false, "second", new(string), nil),
	}}
	cmd := &cobra.Command{Use: "test [a] [b]"}
	AttachArgs(cmd, spec)
	err := cmd.Args(cmd, []string{"one", "two", "three"})
	if err == nil {
		t.Fatal("expected error for extra positional")
	}
	if err := cmd.Args(cmd, []string{"one", "two"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
