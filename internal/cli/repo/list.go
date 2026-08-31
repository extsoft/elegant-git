package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	"github.com/extsoft/elegant-git/internal/cli/statefmt"
	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/memory/repoid"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	var format string
	var nameOrPath string
	spec := listSpec(&nameOrPath)
	c := &cobra.Command{
		Use:   "list [name-or-path]",
		Short: "List repositories or show the current or named one",
		Long:  "Lists repositories in shared memory. With no name, shows the current repository when inside a git work tree, otherwise lists all. Names current and all are selectors; any other name-or-path prints that repository's details.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
				return err
			}
			selector := nameOrPath
			if selector == "" {
				selector = defaultListSelector()
			}
			if selector == shared.SelectorCurrent && format != "table" {
				if nameOrPath == shared.SelectorCurrent {
					return fmt.Errorf("--format is not supported with current")
				}
				selector = shared.SelectorAll
			}
			w := cmd.OutOrStdout()
			if selector == shared.SelectorCurrent {
				return printRepoState(w)
			}
			s, err := shared.Load()
			if err != nil {
				return err
			}
			if selector == shared.SelectorAll {
				return listRepositories(w, s, format)
			}
			_, repo, err := shared.ResolveRepository(s, selector)
			if err != nil {
				return err
			}
			return showRepository(w, s, repo, format)
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

func listSpec(nameOrPath *string) argspec.Spec {
	in := argspec.PositionalInputWithComplete("name-or-path", 0, false, "Repository name or path", nameOrPath, nil, listNameChoices, true)
	in.OmitInteractive = true
	return argspec.Spec{Inputs: []argspec.Input{in}}
}

func listNameChoices(ctx context.Context) ([]argspec.Choice, error) {
	out := []argspec.Choice{
		{Value: shared.SelectorAll, Description: "All managed repositories"},
		{Value: shared.SelectorCurrent, Description: "The current repository"},
	}
	names, err := sources.Repositories(ctx)
	if err != nil {
		return out, nil
	}
	return append(out, names...), nil
}

func printRepoState(w io.Writer) error {
	gitDir, gitErr := memrepo.GitDir()
	if gitErr != nil {
		fmt.Fprintln(w, "repository: (not inside a git work tree)")
		return nil
	}

	s, err := shared.Load()
	if err != nil {
		return err
	}

	repoPath := memrepo.Path(gitDir)
	statefmt.PrintHeading(w, "", "Per-repo memory")
	fmt.Fprintf(w, "  path: %s\n", statefmt.FileStatusLine(repoPath))
	if v := os.Getenv("ELEGANT_GIT_REPO_STATE_FILE"); v != "" {
		fmt.Fprintf(w, "  (ELEGANT_GIT_REPO_STATE_FILE=%s)\n", v)
	}

	repoID, _ := repoid.ReadLocal()
	if repoID != "" {
		fmt.Fprintf(w, "  elegant-git.repo-id: %s\n", repoID)
		if reg, err := shared.GetRepo(s, repoID); err == nil {
			fmt.Fprintf(w, "  registry name: %s\n", reg.Name)
			fmt.Fprintf(w, "  registry path: %s\n", reg.CurrentPath)
			if reg.OriginURL != "" {
				fmt.Fprintf(w, "  origin: %s\n", reg.OriginURL)
			}
			if reg.WorkspaceID == "" {
				fmt.Fprintln(w, "  linked workspace: (not linked)")
			} else if prof, err := shared.GetWorkspace(s, reg.WorkspaceID); err == nil {
				fmt.Fprintf(w, "  linked workspace: %s\n", prof.Name)
			} else {
				fmt.Fprintf(w, "  workspace id: %s (missing from shared memory)\n", reg.WorkspaceID)
			}
		}
	} else {
		fmt.Fprintln(w, "  elegant-git.repo-id: (not set; run repo configure)")
	}

	perRepo, err := memrepo.Load(gitDir)
	registryWorkspaceID := ""
	if repoID != "" {
		if reg, err := shared.GetRepo(s, repoID); err == nil {
			registryWorkspaceID = reg.WorkspaceID
		}
	}
	if err == nil {
		if perRepo.RepoID != "" && perRepo.RepoID != repoID {
			fmt.Fprintf(w, "  memory repo_id: %s\n", perRepo.RepoID)
		}
		if perRepo.WorkspaceID != "" && perRepo.WorkspaceID != registryWorkspaceID {
			if prof, err := shared.GetWorkspace(s, perRepo.WorkspaceID); err == nil {
				fmt.Fprintf(w, "  per-repo memory workspace: %s\n", prof.Name)
			} else {
				fmt.Fprintf(w, "  per-repo memory workspace_id: %s\n", perRepo.WorkspaceID)
			}
		}
		if perRepo.DefaultBranch != "" {
			fmt.Fprintf(w, "  default branch: %s\n", perRepo.DefaultBranch)
		}
		if len(perRepo.ProtectedBranches) > 0 {
			fmt.Fprintf(w, "  protected branches: %v\n", perRepo.ProtectedBranches)
		}
	}

	fmt.Fprintln(w)
	statefmt.PrintHeading(w, "", "Local git identity")
	statefmt.PrintGitIdentity(w, "  ", git.ConfigLocalGet)

	steps := []statefmt.Step{
		{Command: "eg repo list all", Comment: "every managed repository"},
	}
	if repoID != "" {
		if reg, err := shared.GetRepo(s, repoID); err == nil {
			if prof, err := shared.GetWorkspace(s, reg.WorkspaceID); err == nil {
				steps = append(steps, statefmt.Step{
					Command: "eg workspace list " + prof.Name,
					Comment: "the linked workspace",
				})
			}
		}
	}
	statefmt.PrintFurtherSteps(w, steps)
	return nil
}

func listRepositories(w io.Writer, s *shared.State, format string) error {
	if format == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(shared.ListRepos(s))
	}
	var repos []*shared.Repository
	for _, r := range shared.ListRepos(s) {
		if r != nil {
			repos = append(repos, r)
		}
	}
	sort.Slice(repos, func(i, j int) bool { return repos[i].Name < repos[j].Name })
	var items []statefmt.Item
	for _, r := range repos {
		items = append(items, statefmt.Item{
			Heading: r.Name,
			Fields: []statefmt.Field{
				{Key: "workspace", Value: statefmt.WorkspaceName(s, r.WorkspaceID)},
				{Key: "path", Value: r.CurrentPath},
				{Key: "explore", Value: "eg repo list " + r.Name},
			},
		})
	}
	statefmt.PrintCatalog(w, "", items)
	return nil
}

func showRepository(w io.Writer, s *shared.State, r *shared.Repository, format string) error {
	if format == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	}
	fields := []statefmt.Field{
		{Key: "name", Value: r.Name},
		{Key: "path", Value: r.CurrentPath},
	}
	if r.OriginURL != "" {
		fields = append(fields, statefmt.Field{Key: "origin", Value: r.OriginURL})
	}
	statefmt.PrintFields(w, "", fields)
	if len(r.PathHistory) > 0 {
		fmt.Fprintln(w)
		statefmt.PrintHeading(w, "", "Previous paths")
		for _, p := range r.PathHistory {
			fmt.Fprintf(w, "  - %s\n", p)
		}
	}
	fmt.Fprintln(w)
	if r.WorkspaceID == "" {
		fmt.Fprintln(w, "workspace: (not linked)")
	} else if prof, err := shared.GetWorkspace(s, r.WorkspaceID); err == nil {
		statefmt.PrintHeading(w, "", "Workspace")
		statefmt.PrintFields(w, "  ", []statefmt.Field{
			{Key: "name", Value: prof.Name},
			{Key: "user.name", Value: prof.UserName},
			{Key: "user.email", Value: prof.UserEmail},
			{Key: "signing key", Value: text.OrUnset(prof.SigningKey)},
			{Key: "gpg program", Value: text.OrUnset(prof.GPGProgram)},
			{Key: "editor", Value: text.OrUnset(prof.Editor)},
		})
	} else {
		fmt.Fprintln(w, "workspace: (missing from shared memory)")
	}
	fmt.Fprintln(w)
	gitDir := filepath.Join(r.CurrentPath, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		fmt.Fprintln(w, "per-repo memory: (path not accessible)")
		printRepoFurtherSteps(w, s, r)
		return nil
	}
	perRepo, err := memrepo.Load(gitDir)
	if err != nil {
		return err
	}
	statefmt.PrintHeading(w, "", "Per-repo memory")
	memFields := []statefmt.Field{
		{Key: "path", Value: memrepo.Path(gitDir)},
	}
	if perRepo.DefaultBranch != "" {
		memFields = append(memFields, statefmt.Field{Key: "default branch", Value: perRepo.DefaultBranch})
	}
	if len(perRepo.ProtectedBranches) > 0 {
		memFields = append(memFields, statefmt.Field{Key: "protected branches", Value: fmt.Sprintf("%v", perRepo.ProtectedBranches)})
	}
	if len(memFields) > 0 {
		statefmt.PrintFields(w, "  ", memFields)
	}
	printRepoFurtherSteps(w, s, r)
	return nil
}

func printRepoFurtherSteps(w io.Writer, s *shared.State, r *shared.Repository) {
	steps := []statefmt.Step{
		{Command: "eg repo list all", Comment: "every managed repository"},
	}
	if r != nil {
		if prof, err := shared.GetWorkspace(s, r.WorkspaceID); err == nil {
			steps = append(steps, statefmt.Step{
				Command: "eg workspace list " + prof.Name,
				Comment: "the linked workspace",
			})
		}
	}
	statefmt.PrintFurtherSteps(w, steps)
}
