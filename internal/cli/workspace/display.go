package workspace

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/extsoft/elegant-git/internal/cli/statefmt"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/memory/repoid"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/text"
)

// PrintWorkspaceStatus prints the linked workspace for the current repository.
func PrintWorkspaceStatus(w io.Writer) error {
	gitDir, gitErr := memrepo.GitDir()
	if gitErr != nil {
		fmt.Fprintln(w, "repository: (not inside a git work tree)")
		return nil
	}

	s, err := shared.Load()
	if err != nil {
		return err
	}

	repoID, _ := repoid.ReadLocal()
	if repoID == "" {
		fmt.Fprintln(w, "linked workspace: (not set; run repo configure)")
		return nil
	}

	reg, err := shared.GetRepo(s, repoID)
	if err != nil {
		fmt.Fprintf(w, "linked workspace: repo-id %s not in registry\n", repoID)
		return nil
	}

	if reg.WorkspaceID == "" {
		fmt.Fprintln(w, "linked workspace: (not linked)")
		return nil
	}

	prof, err := shared.GetWorkspace(s, reg.WorkspaceID)
	if err != nil {
		fmt.Fprintf(w, "linked workspace: id %s (missing from shared memory)\n", reg.WorkspaceID)
		return nil
	}

	PrintFields(w, "", reg.WorkspaceID, prof)
	PrintLinkedRepos(w, s, prof)

	perRepo, err := memrepo.Load(gitDir)
	if err != nil {
		return err
	}
	if perRepo.WorkspaceID != "" && perRepo.WorkspaceID != reg.WorkspaceID {
		fmt.Fprintln(w)
		if alt, err := shared.GetWorkspace(s, perRepo.WorkspaceID); err == nil {
			statefmt.PrintHeading(w, "", "Per-repo memory workspace")
			PrintFields(w, "  ", perRepo.WorkspaceID, alt)
		} else {
			fmt.Fprintf(w, "per-repo memory workspace_id: %s\n", perRepo.WorkspaceID)
		}
	}
	printWorkspaceFurtherSteps(w, true)
	return nil
}

func printWorkspaceFurtherSteps(w io.Writer, inWorkTree bool) {
	steps := []statefmt.Step{
		{Command: "eg workspace list all", Comment: "every workspace"},
	}
	if inWorkTree {
		steps = append(steps, statefmt.Step{Command: "eg repo list", Comment: "this repository"})
	}
	statefmt.PrintFurtherSteps(w, steps)
}

// PrintFields writes workspace identity fields to w.
func PrintFields(w io.Writer, indent, id string, p *shared.Workspace) {
	namespaces := "(none)"
	if len(p.Namespaces) > 0 {
		namespaces = strings.Join(p.Namespaces, ", ")
	}
	statefmt.PrintFields(w, indent, []statefmt.Field{
		{Key: "name", Value: p.Name},
		{Key: "id", Value: id},
		{Key: "user.name", Value: p.UserName},
		{Key: "user.email", Value: p.UserEmail},
		{Key: "signing key", Value: text.OrUnset(p.SigningKey)},
		{Key: "gpg program", Value: text.OrUnset(p.GPGProgram)},
		{Key: "editor", Value: text.OrUnset(p.Editor)},
		{Key: "namespaces", Value: namespaces},
	})
}

// PrintLinkedRepos writes the workspace's linked repositories as nested catalog blocks.
func PrintLinkedRepos(w io.Writer, s *shared.State, p *shared.Workspace) {
	fmt.Fprintln(w)
	if len(p.LinkedRepos) == 0 {
		statefmt.PrintHeading(w, "", "Linked repositories")
		fmt.Fprintln(w, "  (none)")
		return
	}
	statefmt.PrintHeading(w, "", "Linked repositories")
	var items []statefmt.Item
	for _, repoID := range p.LinkedRepos {
		repo, err := shared.GetRepo(s, repoID)
		if err != nil {
			items = append(items, statefmt.Item{
				Heading: repoID,
				Fields:  []statefmt.Field{{Key: "explore", Value: "eg repo list " + repoID}},
			})
			continue
		}
		items = append(items, statefmt.Item{
			Heading: repo.Name,
			Fields: []statefmt.Field{
				{Key: "path", Value: repo.CurrentPath},
				{Key: "explore", Value: "eg repo list " + repo.Name},
			},
		})
	}
	statefmt.PrintCatalog(w, "  ", items)
}

// PrintList writes all workspaces as a table or JSON.
func PrintList(w io.Writer, s *shared.State, format string) error {
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
	var items []statefmt.Item
	for _, r := range rows {
		items = append(items, statefmt.Item{
			Heading: r.p.Name,
			Fields: []statefmt.Field{
				{Key: "identity", Value: fmt.Sprintf("%s <%s>", r.p.UserName, r.p.UserEmail)},
				{Key: "repositories", Value: fmt.Sprintf("%d", len(r.p.LinkedRepos))},
				{Key: "explore", Value: "eg workspace list " + r.p.Name},
			},
		})
	}
	statefmt.PrintCatalog(w, "", items)
	return nil
}

// PrintDetails writes one workspace's details as a table or JSON.
func PrintDetails(w io.Writer, s *shared.State, name, format string) error {
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
	PrintFields(w, "", id, p)
	PrintLinkedRepos(w, s, p)
	printWorkspaceFurtherSteps(w, false)
	return nil
}
