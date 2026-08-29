package workspace

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

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
	PrintLinkedRepos(w, s, prof, "")

	perRepo, err := memrepo.Load(gitDir)
	if err != nil {
		return err
	}
	if perRepo.WorkspaceID != "" && perRepo.WorkspaceID != reg.WorkspaceID {
		if alt, err := shared.GetWorkspace(s, perRepo.WorkspaceID); err == nil {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "per-repo memory workspace:")
			PrintFields(w, "  ", perRepo.WorkspaceID, alt)
		} else {
			fmt.Fprintf(w, "\nper-repo memory workspace_id: %s\n", perRepo.WorkspaceID)
		}
	}
	return nil
}

// PrintFields writes workspace identity fields to w.
func PrintFields(w io.Writer, prefix, id string, p *shared.Workspace) {
	fmt.Fprintf(w, "%sname:         %s\n", prefix, p.Name)
	fmt.Fprintf(w, "%sid:           %s\n", prefix, id)
	fmt.Fprintf(w, "%suser.name:    %s\n", prefix, p.UserName)
	fmt.Fprintf(w, "%suser.email:   %s\n", prefix, p.UserEmail)
	fmt.Fprintf(w, "%ssigning key:  %s\n", prefix, text.OrUnset(p.SigningKey))
	fmt.Fprintf(w, "%sgpg program:  %s\n", prefix, text.OrUnset(p.GPGProgram))
	fmt.Fprintf(w, "%seditor:       %s\n", prefix, text.OrUnset(p.Editor))
	if len(p.Namespaces) == 0 {
		fmt.Fprintf(w, "%snamespaces:   (none)\n", prefix)
	} else {
		fmt.Fprintf(w, "%snamespaces:   %s\n", prefix, strings.Join(p.Namespaces, ", "))
	}
}

// PrintLinkedRepos writes the workspace's linked repository list to w.
func PrintLinkedRepos(w io.Writer, s *shared.State, p *shared.Workspace, prefix string) {
	if len(p.LinkedRepos) == 0 {
		fmt.Fprintf(w, "%slinked repos: (none)\n", prefix)
		return
	}
	fmt.Fprintf(w, "%slinked repos: %d\n", prefix, len(p.LinkedRepos))
	for _, repoID := range p.LinkedRepos {
		repo, err := shared.GetRepo(s, repoID)
		if err != nil {
			fmt.Fprintf(w, "%s  - %s\n", prefix, repoID)
			continue
		}
		fmt.Fprintf(w, "%s  - %s    %s\n", prefix, repo.Name, repo.CurrentPath)
	}
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
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s <%s>\t%d repo(s)\n", r.p.Name, r.p.UserName, r.p.UserEmail, len(r.p.LinkedRepos))
	}
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
	PrintLinkedRepos(w, s, p, "")
	return nil
}
