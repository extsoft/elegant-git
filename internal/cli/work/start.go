package work

import (
	"context"
	"fmt"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/cli/completion"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cli/sources"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/pipe"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/bees-hive/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

var startID = cmdid.ID{Command: "work", Action: "start"}

var startChangeOptions = []string{"add", "reset", "cancel"}

func newStartCommand() *cobra.Command {
	var name, fromRef string
	spec := startSpec(&name, &fromRef)
	c := &cobra.Command{
		Use:   "start <name> [from-ref]",
		Short: "Creates a new branch",
		Long: `Creates a new local branch from the default development branch (or from-ref).

When there are uncommitted changes, you can add them to the new branch or reset them before checkout.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, startID, func() error {
				return startRun(cmd, args, spec)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func startSpec(name, fromRef *string) argspec.Spec {
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInput("name", 0, true, "Branch name", name, nil),
		argspec.PositionalInputWithComplete("from-ref", 1, false, "Start from ref", fromRef, config.DefaultBranch, sources.Refs, true),
	}}
}

func startRun(cmd *cobra.Command, args []string, spec argspec.Spec) error {
	if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
		return err
	}
	return startRunWithRefs(cmd.Context(), spec.Inputs[0].Get(), spec.Inputs[1].Get())
}

func startRunWithRefs(ctx context.Context, name, fromRef string) error {
	target := config.DefaultBranch()
	if fromRef != "" {
		target = fromRef
	}
	logic := func() error {
		if state.AreThereRemotes() {
			cliruntime.FetchOrInform()
		}
		startPoint := resolveStartPoint(target)
		if err := git.Verbose("checkout", "-b", name, "--no-track", startPoint); err != nil {
			return err
		}
		return config.SetBranchSourceBranch(name, target)
	}
	mode, err := resolveStartChanges(ctx)
	if err != nil {
		return err
	}
	switch mode {
	case "none":
		return logic()
	case "stash":
		return pipe.StashPipe(startID, logic)
	case "reset":
		if err := git.Verbose("reset", "--hard", "HEAD"); err != nil {
			return err
		}
		return logic()
	case "cancel":
		return fmt.Errorf("work start cancelled")
	default:
		return fmt.Errorf("invalid choice %q (use add, reset, or cancel)", mode)
	}
}

func resolveStartChanges(ctx context.Context) (string, error) {
	if !pipe.HasChanges() {
		return "none", nil
	}
	p := prompt.FromContext(ctx)
	if prompt.NonInteractive(p) {
		return "stash", nil
	}
	ans, err := p.Closed("There are uncommitted changes", startChangeOptions, "add", true)
	if err != nil {
		return "", err
	}
	mode, ok := startChangeMode(ans)
	if !ok {
		return "", fmt.Errorf("invalid choice %q (use add, reset, or cancel)", ans)
	}
	return mode, nil
}

func resolveStartPoint(target string) string {
	if state.IsThereUpstreamFor(target) {
		if upstream := state.UpstreamOf(target); upstream != "" {
			return upstream
		}
	}
	return target
}

func parseStartChangesChoice(answer string) (string, bool) {
	got, ok := prompt.MatchClosed(answer, startChangeOptions, "add")
	if !ok {
		return "", false
	}
	return startChangeMode(got)
}

func startChangeMode(ans string) (string, bool) {
	switch ans {
	case "add":
		return "stash", true
	case "reset":
		return "reset", true
	case "cancel":
		return "cancel", true
	default:
		return "", false
	}
}
