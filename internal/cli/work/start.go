package work

import (
	"context"
	"fmt"
	"strings"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/cli/completion"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cli/sources"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/pipe"
	"github.com/bees-hive/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

var startID = cmdid.ID{Command: "work", Action: "start"}

const startChangesPrompt = `There are uncommitted changes.
[a] Add them to the new branch (default)
[r] Reset (discard) local changes
[c] Cancel
Choice [a/r/c]: `

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
		if err := git.Verbose("checkout", target); err != nil {
			return err
		}
		if state.IsThereUpstreamFor(target) {
			cliruntime.PullOrInform()
		}
		return git.Verbose("checkout", "-b", name)
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
		return fmt.Errorf("invalid choice %q (use a, r, or c)", mode)
	}
}

func resolveStartChanges(ctx context.Context) (string, error) {
	if !pipe.HasChanges() {
		return "none", nil
	}
	if !cliruntime.StdinIsInteractive(ctx) {
		return "stash", nil
	}
	answer, err := cliruntime.ReadLineAnswer(ctx, startChangesPrompt)
	if err != nil {
		return "", err
	}
	mode, ok := parseStartChangesChoice(answer)
	if !ok {
		return "", fmt.Errorf("invalid choice %q (use a, r, or c)", answer)
	}
	return mode, nil
}

func parseStartChangesChoice(answer string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "", "a", "add", "keep", "move":
		return "stash", true
	case "r", "reset", "discard":
		return "reset", true
	case "c", "cancel", "quit", "q":
		return "cancel", true
	default:
		return "", false
	}
}
