package workspace

import (
	"fmt"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func linkSpec(name *string) argspec.Spec {
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInputWithComplete("name", 0, true, "Workspace name", name, nil, sources.Workspaces, true),
	}}
}

func newLinkCommand() *cobra.Command {
	var name string
	spec := linkSpec(&name)
	c := &cobra.Command{
		Use:   "link <name>",
		Short: "Link the current repository to a workspace",
		Long:  "Links the current repository to an existing workspace, applies that workspace's identity, and may capture the origin namespace.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
				return err
			}
			return linkRun(cmd, name)
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func linkRun(cmd *cobra.Command, name string) error {
	if _, err := memrepo.GitDir(); err != nil {
		return fmt.Errorf("not a git repository")
	}
	s, err := shared.Load()
	if err != nil {
		return err
	}
	id, ws, err := shared.GetWorkspaceByName(s, name)
	if err != nil {
		return err
	}
	applied, err := applyToCurrentRepo(cmd, s, id, ws)
	if err != nil {
		return err
	}
	if !applied {
		return nil
	}
	if err := shared.Save(s); err != nil {
		return err
	}
	text.InfoText(fmt.Sprintf("Linked repository to workspace %s", ws.Name))
	return nil
}
