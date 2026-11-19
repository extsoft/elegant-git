package completion

import (
	"context"
	"testing"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/git"
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

func TestGitBackedCompletionRefusesBeforeProvider(t *testing.T) {
	git.SetVersionProbeForTest(func(string) (string, error) {
		return "git version 2.39.0", nil
	})
	t.Cleanup(func() { git.SetVersionProbeForTest(nil) })
	called := false
	fn := func(context.Context) ([]argspec.Choice, error) {
		called = true
		return []argspec.Choice{{Value: "main"}}, nil
	}
	var ref string
	cmd := &cobra.Command{Use: "switch"}
	cmd.Flags().String("branch", "", "")
	Attach(cmd, argspec.Spec{Inputs: []argspec.Input{
		argspec.GitBacked(argspec.PositionalInputWithComplete("ref", 0, true, "Ref", &ref, nil, fn, false)),
		argspec.GitBacked(argspec.Input{Kind: argspec.Flag, FlagName: "branch", Complete: fn}),
	}})
	if _, dir := cmd.ValidArgsFunction(cmd, nil, ""); dir != cobra.ShellCompDirectiveError || called {
		t.Fatalf("positional dir=%v called=%v", dir, called)
	}
	flagFn, ok := cmd.GetFlagCompletionFunc("branch")
	if !ok {
		t.Fatal("missing flag completion")
	}
	if _, dir := flagFn(cmd, nil, ""); dir != cobra.ShellCompDirectiveError || called {
		t.Fatalf("flag dir=%v called=%v", dir, called)
	}
}

func TestStaticCompletionDoesNotProbeGit(t *testing.T) {
	git.SetVersionProbeForTest(func(string) (string, error) {
		t.Fatal("static completion probed git")
		return "", nil
	})
	t.Cleanup(func() { git.SetVersionProbeForTest(nil) })
	var shell string
	cmd := &cobra.Command{Use: "completion"}
	Attach(cmd, argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInputWithComplete("shell", 0, true, "Shell", &shell, nil, func(context.Context) ([]argspec.Choice, error) {
			return []argspec.Choice{{Value: "bash"}}, nil
		}, true),
	}})
	out, dir := cmd.ValidArgsFunction(cmd, nil, "")
	if dir != cobra.ShellCompDirectiveNoFileComp || len(out) != 1 || out[0] != "bash" {
		t.Fatalf("dir=%v out=%v", dir, out)
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
