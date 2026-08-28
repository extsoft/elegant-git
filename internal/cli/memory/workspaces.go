package memory

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/cli/completion"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cli/sources"
	"github.com/bees-hive/elegant-git/internal/deprecation"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/spf13/cobra"
)

func newWorkspacesCommand() *cobra.Command {
	return newWorkspacesCommandNamed("workspaces")
}

func newLegacyProfilesCommand() *cobra.Command {
	c := newWorkspacesCommandNamed("profiles")
	c.Hidden = true
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[deprecation.SurfaceAnnotation] = "memory profiles"
	return c
}

func newWorkspacesCommandNamed(use string) *cobra.Command {
	var format string
	var name string
	spec := workspacesSpec(&name)
	c := &cobra.Command{
		Use:   use + " [name]",
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
				return showWorkspace(w, s, name, format)
			}
			return listWorkspaces(w, s, format)
		},
	}
	c.Flags().StringVar(&format, "format", "table", "output format: table or json")
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func workspacesSpec(name *string) argspec.Spec {
	in := argspec.PositionalInputWithComplete("name", 0, false, "Workspace name", name, nil, sources.Workspaces, true)
	in.OmitInteractive = true
	return argspec.Spec{Inputs: []argspec.Input{in}}
}

func listWorkspaces(w io.Writer, s *shared.State, format string) error {
	if format == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(shared.ListWorkspaces(s))
	}
	type row struct {
		id string
		p  *shared.Workspace
	}
	var rows []row
	for id, p := range shared.ListWorkspaces(s) {
		if p != nil {
			rows = append(rows, row{id, p})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].p.Name < rows[j].p.Name })
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s <%s>\t%d repo(s)\n", r.p.Name, r.p.UserName, r.p.UserEmail, len(r.p.LinkedRepos))
	}
	return nil
}

func showWorkspace(w io.Writer, s *shared.State, name, format string) error {
	id, p, err := shared.GetWorkspaceByName(s, name)
	if err != nil {
		return err
	}
	if format == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"id":        id,
			"workspace": p,
		})
	}
	printWorkspaceFields(w, "", id, p)
	printLinkedRepos(w, s, p, "")
	return nil
}
