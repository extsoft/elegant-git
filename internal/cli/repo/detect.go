package repo

import (
	"github.com/extsoft/elegant-git/internal/cli/catalog"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/memory/repoid"
	"github.com/extsoft/elegant-git/internal/prompt"
)

type snapshot struct {
	InGit      bool
	Configured bool
}

type outcome struct {
	Steps []string
}

const detectionTitle = "Detection action..."

func inspect() snapshot {
	s := snapshot{}
	if _, err := memrepo.GitDir(); err != nil {
		return s
	}
	s.InGit = true
	id, err := repoid.ReadLocal()
	if err != nil || id == "" {
		return s
	}
	s.Configured = true
	return s
}

func detect(s snapshot) outcome {
	var steps []string
	if s.InGit {
		steps = append(steps, "in a git repository? yes")
		if s.Configured {
			steps = append(steps, "configured? yes")
		} else {
			steps = append(steps, "configured? no")
		}
	} else {
		steps = append(steps, "in a git repository? no")
	}
	steps = append(steps, "selected: ask")
	return outcome{Steps: steps}
}

func askOptions(s snapshot) []string {
	if !s.InGit {
		return []string{"clone", "init", "list", "help", "quit"}
	}
	if !s.Configured {
		return []string{"configure", "list", "prune", "doctor", "help", "quit"}
	}
	return []string{"list", "sync", "prune", "configure", "doctor", "help", "quit"}
}

func askChoices(s snapshot) []prompt.Choice {
	opts := askOptions(s)
	out := make([]prompt.Choice, len(opts))
	for i, action := range opts {
		c := prompt.Choice{Value: action, Description: catalog.Purpose("repo", action)}
		if action != "quit" {
			c.Display = "repo " + action
		}
		out[i] = c
	}
	return out
}
