package memory

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/cli/completion"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	memrepo "github.com/bees-hive/elegant-git/internal/memory/repo"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
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
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.Name, profileName(s, r.ProfileID), r.CurrentPath)
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
	if prof, err := shared.GetProfile(s, r.ProfileID); err == nil {
		fmt.Fprintln(w, "profile:")
		fmt.Fprintf(w, "  name:         %s\n", prof.Name)
		fmt.Fprintf(w, "  user.name:    %s\n", prof.UserName)
		fmt.Fprintf(w, "  user.email:   %s\n", prof.UserEmail)
		fmt.Fprintf(w, "  signing key:  %s\n", showOptional(prof.SigningKey))
		fmt.Fprintf(w, "  gpg program:  %s\n", showOptional(prof.GPGProgram))
		fmt.Fprintf(w, "  editor:       %s\n", showOptional(prof.Editor))
	} else {
		fmt.Fprintln(w, "profile: (missing from shared memory)")
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

func profileName(s *shared.State, profileID string) string {
	if p, err := shared.GetProfile(s, profileID); err == nil {
		return p.Name
	}
	return ""
}
