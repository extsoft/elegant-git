package workspace

import (
	"context"
	"fmt"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	var format string
	var name string
	spec := listSpec(&name)
	c := &cobra.Command{
		Use:   "list [name]",
		Short: "List workspaces or show the current or named one",
		Long:  "Lists workspaces in shared memory. With no name, shows the linked workspace when inside a git work tree, otherwise lists all. Names current and all are selectors; any other name prints that workspace's details.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
				return err
			}
			selector := name
			if selector == "" {
				selector = defaultListSelector()
			}
			if selector == shared.SelectorCurrent && format != "table" {
				if name == shared.SelectorCurrent {
					return fmt.Errorf("--format is not supported with current")
				}
				selector = shared.SelectorAll
			}
			w := cmd.OutOrStdout()
			if selector == shared.SelectorCurrent {
				return PrintWorkspaceStatus(w)
			}
			s, err := shared.Load()
			if err != nil {
				return err
			}
			if selector == shared.SelectorAll {
				return PrintList(w, s, format)
			}
			return PrintDetails(w, s, selector, format)
		},
	}
	c.Flags().StringVar(&format, "format", "table", "output format: table or json (default table)")
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func defaultListSelector() string {
	if _, err := memrepo.GitDir(); err == nil {
		return shared.SelectorCurrent
	}
	return shared.SelectorAll
}

func listSpec(name *string) argspec.Spec {
	in := argspec.PositionalInputWithComplete("name", 0, false, "Workspace name", name, nil, listNameChoices, true)
	in.OmitInteractive = true
	return argspec.Spec{Inputs: []argspec.Input{in}}
}

func listNameChoices(ctx context.Context) ([]argspec.Choice, error) {
	out := []argspec.Choice{
		{Value: shared.SelectorAll, Description: "All workspaces"},
		{Value: shared.SelectorCurrent, Description: "Workspace linked to the current repository"},
	}
	names, err := sources.Workspaces(ctx)
	if err != nil {
		return out, nil
	}
	return append(out, names...), nil
}
