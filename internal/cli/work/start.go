package work

import (
	"strings"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/pipe"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

var startID = cmdid.ID{Command: "work", Action: "start"}

func newStartCommand() *cobra.Command {
	var name, fromRef string
	spec := startSpec(&name, &fromRef)
	c := &cobra.Command{
		Use:   "start <name> [from-ref]",
		Short: "Creates a new branch",
		Long: `Creates a new local branch from the default development branch (or from-ref).

When there are uncommitted changes, choose carry, stash, discard, or cancel. carry moves tracked and non-ignored untracked files onto the new branch and keeps the staged and unstaged split. stash leaves them in the stash instead. discard removes tracked edits only. cancel stops before any of that. A non-interactive run with uncommitted changes fails unless --dirty is set. A clean tree does not need --dirty.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, startID, func() error {
				return startRun(cmd, args, spec)
			})
		},
	}
	addDirtyFlags(c)
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
	p := prompt.FromContext(cmd.Context())
	bindStartArgs(args, spec)
	name, err := ensureStartName(p, spec)
	if err != nil {
		return err
	}
	spec.Inputs[0].Set(name)
	if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
		return err
	}
	return startRunWithRefs(cmd, name, spec.Inputs[1].Get())
}

func bindStartArgs(args []string, spec argspec.Spec) {
	for i := range spec.Inputs {
		in := &spec.Inputs[i]
		if in.Kind == argspec.Positional && in.Index < len(args) && strings.TrimSpace(in.Get()) == "" {
			in.Set(args[in.Index])
		}
	}
}

func ensureStartName(p prompt.Prompter, spec argspec.Spec) (string, error) {
	name := strings.TrimSpace(spec.Inputs[0].Get())
	if name != "" {
		return guardNewBranchName(p, name)
	}
	if prompt.NonInteractive(p) {
		return "", &argspec.ErrMissingRequired{Names: []string{"name"}}
	}
	branches, err := state.LocalBranchNames()
	if err != nil {
		return "", err
	}
	return askNewBranchName(p, "Branch name", branches, "", prepareCaseFold())
}

func startRunWithRefs(cmd *cobra.Command, name, fromRef string) error {
	var source, startPoint string
	load := func() error {
		source = config.DefaultBranch()
		if fromRef != "" {
			source = fromRef
		}
		if state.AreThereRemotes() {
			cliruntime.FetchOrInform()
		}
		startPoint = resolveStartPoint(source)
		return nil
	}
	checkout := func() error {
		if startPoint == "" {
			if err := load(); err != nil {
				return err
			}
		}
		if err := git.Verbose("checkout", "-b", name, "--no-track", startPoint); err != nil {
			return err
		}
		if err := pipe.RememberRestoreBranch(name); err != nil {
			return err
		}
		return config.SetBranchSourceBranch(name, source)
	}
	return applyNavigation(navRequest{
		cmd:  cmd,
		ctx:  cmd.Context(),
		id:   startID,
		verb: "work start",
		dest: name,
		target: func() (string, error) {
			if err := load(); err != nil {
				return "", err
			}
			return diffTarget(startPoint), nil
		},
		body: checkout,
	})
}

func resolveStartPoint(target string) string {
	if state.IsThereUpstreamFor(target) {
		if upstream := state.UpstreamOf(target); upstream != "" {
			return upstream
		}
	}
	return target
}
