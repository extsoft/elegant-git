package workspace

import (
	"fmt"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func deleteSpec(name *string) argspec.Spec {
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInputWithComplete("name", 0, true, "Workspace name", name, nil, sources.Workspaces, true),
	}}
}

func newDeleteCommand() *cobra.Command {
	var name string
	var yes bool
	spec := deleteSpec(&name)
	c := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a workspace",
		Long:  "Deletes a workspace from shared memory. Linked repositories stay in the registry with their workspace_id cleared; their git config and files are left untouched. Pass --yes to skip the confirmation prompt.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
				return err
			}
			s, err := shared.Load()
			if err != nil {
				return err
			}
			id, ws, err := shared.GetWorkspaceByName(s, name)
			if err != nil {
				return err
			}
			p := prompt.FromContext(cmd.Context())
			if err := deleteSummaryConfirm(cmd, s, id, ws, yes, p); err != nil {
				return err
			}
			if err := shared.DeleteWorkspace(s, id); err != nil {
				return err
			}
			if err := shared.Validate(s); err != nil {
				return err
			}
			if err := shared.Save(s); err != nil {
				return err
			}
			text.InfoText("Deleted workspace " + name)
			return nil
		},
	}
	c.Flags().BoolVar(&yes, "yes", false, "skip confirmation prompt")
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func deleteSummaryConfirm(cmd *cobra.Command, s *shared.State, id string, ws *shared.Workspace, yes bool, p prompt.Prompter) error {
	_ = id
	fmt.Fprintf(cmd.OutOrStdout(), "Workspace %q will be deleted.\n", ws.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "  user.name:  %s\n", ws.UserName)
	fmt.Fprintf(cmd.OutOrStdout(), "  user.email: %s\n", ws.UserEmail)

	linked := append([]string(nil), ws.LinkedRepos...)
	if len(linked) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\nUnlinks %d repository (-ies). Their git config and files are left untouched:\n", len(linked))
		for _, repoID := range linked {
			repo, err := shared.GetRepo(s, repoID)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", repoID)
				continue
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s    %s\n", repo.Name, repo.CurrentPath)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Run `git elegant repo configure <workspace>` in each to link it elsewhere.")
	}
	fmt.Fprintln(cmd.OutOrStdout())

	if yes {
		return nil
	}
	if prompt.NonInteractive(p) {
		return fmt.Errorf("confirmation required; pass --yes to delete without prompting")
	}
	defaultYes := len(linked) == 0
	ok, err := p.Confirm("Delete?", defaultYes)
	return errIfNotOK(ok, err)
}
