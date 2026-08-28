package workspace

import (
	memrepo "github.com/bees-hive/elegant-git/internal/memory/repo"
	"github.com/bees-hive/elegant-git/internal/memory/repoid"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/prompt"
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

var workspaceActionPurpose = map[string]string{
	"list":   "Shows available workspaces.",
	"new":    "Creates a workspace.",
	"link":   "Links the current repository to a workspace.",
	"edit":   "Edits a workspace.",
	"delete": "Deletes a workspace.",
	"status": "Shows the linked workspace for the current repository.",
	"fetch":  "Fetches remotes for linked repositories.",
	"quit":   "Leave without another action.",
}

func askOptions(s snapshot) []string {
	hasWS := s.WorkspaceCount > 0
	if !s.InGit {
		opts := []string{"new"}
		if hasWS {
			opts = []string{"list", "new", "edit", "delete"}
		}
		return append(opts, "quit")
	}
	if !s.Linked {
		opts := []string{"new"}
		if hasWS {
			opts = append(opts, "link")
		}
		return append(opts, "quit")
	}
	opts := []string{"new", "status", "fetch"}
	if hasWS {
		opts = []string{"list", "new", "link", "edit", "delete", "status", "fetch"}
	}
	return append(opts, "quit")
}

func askChoices(s snapshot) []prompt.Choice {
	opts := askOptions(s)
	out := make([]prompt.Choice, len(opts))
	for i, action := range opts {
		out[i] = prompt.Choice{Value: action, Description: workspaceActionPurpose[action]}
	}
	return out
}
