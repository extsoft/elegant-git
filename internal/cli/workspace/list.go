package workspace

import (
	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	var format string
	var name string
	spec := listSpec(&name)
	c := &cobra.Command{
		Use:   "list [name]",
		Short: "List workspaces or show one workspace's details",
		Long:  "Lists workspaces in shared memory. When name is given, prints full details for that workspace.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
				return err
			}
			s, err := shared.Load()
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if name != "" {
				return PrintDetails(w, s, name, format)
			}
			return PrintList(w, s, format)
		},
	}
	c.Flags().StringVar(&format, "format", "table", "output format: table or json")
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func listSpec(name *string) argspec.Spec {
	in := argspec.PositionalInputWithComplete("name", 0, false, "Workspace name", name, nil, sources.Workspaces, true)
	in.OmitInteractive = true
	return argspec.Spec{Inputs: []argspec.Input{in}}
}
