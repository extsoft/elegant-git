package completion

import (
	"context"
	"fmt"
	"strings"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/spf13/cobra"
)

// Attach wires shell completion and positional arg limits from argspec on cmd.
func Attach(cmd *cobra.Command, spec argspec.Spec) {
	AttachArgs(cmd, spec)
	positional := positionalByIndex(spec)
	if len(positional) > 0 {
		cmd.ValidArgsFunction = func(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			ctx := c.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			idx := len(args)
			in, ok := positional[idx]
			if !ok || in.Complete == nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			choices, err := in.Complete(ctx)
			return formatCompletion(choices, err, toComplete)
		}
	}
	for _, in := range spec.Inputs {
		if in.Kind != argspec.Flag || in.Complete == nil || in.FlagName == "" {
			continue
		}
		flagName := in.FlagName
		complete := in.Complete
		_ = cmd.RegisterFlagCompletionFunc(flagName, func(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			ctx := c.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			choices, err := complete(ctx)
			return formatCompletion(choices, err, toComplete)
		})
	}
}

// AttachArgs limits positional arguments from an argspec (no completion wiring).
func AttachArgs(cmd *cobra.Command, spec argspec.Spec) {
	maxPos := -1
	for _, in := range spec.Inputs {
		if in.Kind == argspec.Positional && in.Index > maxPos {
			maxPos = in.Index
		}
	}
	if maxPos < 0 {
		cmd.Args = cliruntime.RejectExtraArgs
		return
	}
	limit := maxPos + 1
	cmd.Args = func(c *cobra.Command, args []string) error {
		if len(args) > limit {
			return cliruntime.NewUsageError(c, fmt.Errorf("accepts at most %d positional arg(s), received %d", limit, len(args)))
		}
		return nil
	}
}

func positionalByIndex(spec argspec.Spec) map[int]argspec.Input {
	out := map[int]argspec.Input{}
	for _, in := range spec.Inputs {
		if in.Kind == argspec.Positional && in.Complete != nil {
			out[in.Index] = in
		}
	}
	return out
}

func formatCompletion(choices []argspec.Choice, err error, toComplete string) ([]string, cobra.ShellCompDirective) {
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var out []string
	prefix := toComplete
	for _, c := range choices {
		if prefix != "" && !strings.HasPrefix(c.Value, prefix) {
			continue
		}
		if c.Description != "" {
			out = append(out, c.Value+"\t"+c.Description)
		} else {
			out = append(out, c.Value)
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}
