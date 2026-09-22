package workspace

import (
	"github.com/extsoft/elegant-git/internal/cli/catalog"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/memory/repoid"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
)

type snapshot struct {
	InGit          bool
	Linked         bool
	WorkspaceName  string
	WorkspaceCount int
}

type outcome struct {
	Steps []string
}

const detectionTitle = "Detection action..."

func inspect() snapshot {
	s := snapshot{}
	st, loadErr := shared.Load()
	if loadErr == nil {
		s.WorkspaceCount = len(st.Workspaces)
	}
	if _, err := memrepo.GitDir(); err != nil {
		return s
	}
	s.InGit = true
	if loadErr != nil {
		return s
	}
	repoID, err := repoid.ReadLocal()
	if err != nil || repoID == "" {
		return s
	}
	reg, err := shared.GetRepo(st, repoID)
	if err != nil || reg.WorkspaceID == "" {
		return s
	}
	ws, err := shared.GetWorkspace(st, reg.WorkspaceID)
	if err != nil {
		return s
	}
	s.Linked = true
	s.WorkspaceName = ws.Name
	return s
}

func detect(s snapshot) outcome {
	var steps []string
	if s.InGit {
		steps = append(steps, "in a git repository? yes")
		if s.Linked {
			steps = append(steps, "workspace linked? yes ("+s.WorkspaceName+")")
		} else {
			steps = append(steps, "workspace linked? no")
		}
	} else {
		steps = append(steps, "in a git repository? no")
	}
	steps = append(steps, "selected: ask")
	return outcome{Steps: steps}
}

func askOptions(s snapshot) []string {
	hasWS := s.WorkspaceCount > 0
	if !s.InGit {
		opts := []string{"new"}
		if hasWS {
			opts = []string{"list", "new", "edit", "delete", "doctor"}
		}
		return append(opts, "help", "quit")
	}
	if !s.Linked {
		opts := []string{"new"}
		if hasWS {
			opts = append(opts, "link", "doctor")
		}
		return append(opts, "help", "quit")
	}
	opts := []string{"new", "list", "fetch"}
	if hasWS {
		opts = []string{"list", "new", "link", "edit", "delete", "fetch", "doctor"}
	}
	return append(opts, "help", "quit")
}

func askChoices(s snapshot) []prompt.Choice {
	opts := askOptions(s)
	out := make([]prompt.Choice, len(opts))
	for i, action := range opts {
		c := prompt.Choice{Value: action, Description: catalog.Purpose("workspace", action)}
		if action != "quit" {
			c.Display = "workspace " + action
		}
		out[i] = c
	}
	return out
}
