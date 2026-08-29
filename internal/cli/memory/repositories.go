package memory

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newRepositoriesCommand() *cobra.Command {
	var nameOrPath string
	spec := repositoriesSpec(&nameOrPath)
	c := &cobra.Command{
		Use:   "repositories [name-or-path]",
		Short: "List managed repositories or show one repository's details",
		Long:  "Lists managed repositories in shared memory. When name-or-path is given, prints full details for that repository.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
				return err
			}
			s, err := shared.Load()
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if nameOrPath != "" {
				_, repo, err := shared.ResolveRepository(s, nameOrPath)
				if err != nil {
					return err
				}
				return showRepository(w, s, repo)
			}
			return listRepositories(w, s)
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func repositoriesSpec(nameOrPath *string) argspec.Spec {
	in := argspec.PositionalInput("name-or-path", 0, false, "Repository name or path", nameOrPath, nil)
	in.OmitInteractive = true
	return argspec.Spec{Inputs: []argspec.Input{in}}
}

func listRepositories(w io.Writer, s *shared.State) error {
	var repos []*shared.Repository
	for _, r := range shared.ListRepos(s) {
		if r != nil {
			repos = append(repos, r)
		}
	}
	sort.Slice(repos, func(i, j int) bool { return repos[i].Name < repos[j].Name })
	for _, r := range repos {
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.Name, workspaceName(s, r.WorkspaceID), r.CurrentPath)
	}
	return nil
}

func showRepository(w io.Writer, s *shared.State, r *shared.Repository) error {
	fmt.Fprintf(w, "name:  %s\n", r.Name)
	fmt.Fprintf(w, "path:  %s\n", r.CurrentPath)
	if r.OriginURL != "" {
		fmt.Fprintf(w, "origin: %s\n", r.OriginURL)
	}
	if len(r.PathHistory) > 0 {
		fmt.Fprintln(w, "previous paths:")
		for _, p := range r.PathHistory {
			fmt.Fprintf(w, "  - %s\n", p)
		}
	}
	if r.WorkspaceID == "" {
		fmt.Fprintln(w, "workspace: (not linked)")
	} else if prof, err := shared.GetWorkspace(s, r.WorkspaceID); err == nil {
		fmt.Fprintln(w, "workspace:")
		fmt.Fprintf(w, "  name:         %s\n", prof.Name)
		fmt.Fprintf(w, "  user.name:    %s\n", prof.UserName)
		fmt.Fprintf(w, "  user.email:   %s\n", prof.UserEmail)
		fmt.Fprintf(w, "  signing key:  %s\n", text.OrUnset(prof.SigningKey))
		fmt.Fprintf(w, "  gpg program:  %s\n", text.OrUnset(prof.GPGProgram))
		fmt.Fprintf(w, "  editor:       %s\n", text.OrUnset(prof.Editor))
	} else {
		fmt.Fprintln(w, "workspace: (missing from shared memory)")
	}
	gitDir := filepath.Join(r.CurrentPath, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		fmt.Fprintln(w, "per-repo memory: (path not accessible)")
		return nil
	}
	perRepo, err := memrepo.Load(gitDir)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "per-repo memory: %s\n", memrepo.Path(gitDir))
	if perRepo.DefaultBranch != "" {
		fmt.Fprintf(w, "  default branch: %s\n", perRepo.DefaultBranch)
	}
	if len(perRepo.ProtectedBranches) > 0 {
		fmt.Fprintf(w, "  protected branches: %v\n", perRepo.ProtectedBranches)
	}
	return nil
}

func workspaceName(s *shared.State, workspaceID string) string {
	if workspaceID == "" {
		return "(none)"
	}
	if p, err := shared.GetWorkspace(s, workspaceID); err == nil {
		return p.Name
	}
	return ""
}
