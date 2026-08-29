package work

import (
	"fmt"
	"strconv"
	"strings"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/pipe"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/state"
)

type snapshot struct {
	Branch         string
	Detached       bool
	Protected      bool
	Dirty          bool
	Rebasing       bool
	RebasingBranch string
	HasUpstream    bool
	Ahead          int
	Behind         int
	UniqueCommits  bool
	Remotes        bool
}

type outcome struct {
	Action  string
	ThenAsk bool
	Steps   []string
}

func inspect() snapshot {
	branch := cliruntime.CurrentBranch()
	detached := branch == "HEAD"
	s := snapshot{
		Branch:    branch,
		Detached:  detached,
		Protected: !detached && config.IsBranchProtected(branch),
		Dirty:     pipe.HasChanges(),
		Remotes:   state.AreThereRemotes(),
	}
	if state.IsThereActiveRebase() {
		s.Rebasing = true
		s.RebasingBranch = state.RebasingBranch()
	}
	if detached {
		return s
	}
	if state.IsThereUpstreamFor(branch) {
		s.HasUpstream = true
		s.Ahead, s.Behind = aheadBehind()
	}
	latest := config.FreshestBranchSourceBranch(branch)
	s.UniqueCommits = strings.TrimSpace(git.OutputOK("rev-list", latest+".."+branch)) != ""
	return s
}

func aheadBehind() (int, int) {
	out := strings.TrimSpace(git.OutputOK("rev-list", "--left-right", "--count", "HEAD...@{upstream}"))
	fields := strings.Fields(out)
	if len(fields) < 2 {
		return 0, 0
	}
	ahead, _ := strconv.Atoi(fields[0])
	behind, _ := strconv.Atoi(fields[1])
	return ahead, behind
}

const detectionTitle = "Detection action..."

func decisionLine(action string) string {
	if action == "" {
		return "selected: ask"
	}
	return "selected: eg work " + action
}

func detect(s snapshot) outcome {
	var steps []string
	decide := func(action string, thenAsk bool) outcome {
		steps = append(steps, decisionLine(action))
		return outcome{Action: action, ThenAsk: thenAsk, Steps: steps}
	}

	if s.Rebasing {
		if s.RebasingBranch == acceptWorkBranch {
			steps = append(steps, "rebase in progress? yes ("+acceptWorkBranch+")")
			return decide("accept", false)
		}
		if s.RebasingBranch != "" {
			steps = append(steps, "rebase in progress? yes ("+s.RebasingBranch+")")
		} else {
			steps = append(steps, "rebase in progress? yes")
		}
		return decide("polish", false)
	}
	steps = append(steps, "rebase in progress? no")

	prot := "no"
	if s.Protected {
		prot = "yes"
	}
	steps = append(steps, fmt.Sprintf("on protected branch '%s'? %s", s.Branch, prot))

	if s.Dirty {
		steps = append(steps, "uncommitted changes? yes")
		if !s.Detached {
			if s.Protected {
				return decide("start", false)
			}
			return decide("save", false)
		}
		return decide("", true)
	}
	steps = append(steps, "uncommitted changes? no")

	if s.HasUpstream && s.Ahead == 0 && s.Behind > 0 {
		steps = append(steps, behindLine(s, true))
		return decide("sync", true)
	}
	steps = append(steps, behindLine(s, false))

	if evenOrNoUpstream(s) && !s.UniqueCommits {
		steps = append(steps, uniqueLine(s))
		return decide("list", true)
	}
	steps = append(steps, uniqueLine(s))
	return decide("", true)
}

func behindLine(s snapshot, matched bool) string {
	if !s.HasUpstream {
		return "behind upstream only? no (no upstream)"
	}
	yn := "no"
	if matched {
		yn = "yes"
	}
	return fmt.Sprintf("behind upstream only? %s (ahead %d, behind %d)", yn, s.Ahead, s.Behind)
}

func uniqueLine(s snapshot) string {
	yn := "no"
	if s.UniqueCommits {
		yn = "yes"
	}
	return "unique commits vs source? " + yn
}

func evenOrNoUpstream(s snapshot) bool {
	if !s.HasUpstream {
		return true
	}
	return s.Ahead == 0 && s.Behind == 0
}

var workActionPurpose = map[string]string{
	"start":  "Creates a new branch.",
	"save":   "Commits current modifications.",
	"amend":  "Amends the last commit.",
	"list":   "Prints HEAD state.",
	"polish": "Rebases HEAD interactively.",
	"sync":   "Actualizes the branch with upstream commits.",
	"push":   "Publishes HEAD to a remote repository.",
	"track":  "Checks out a remote-tracking branch.",
	"accept": "Adds modifications to the default development branch.",
	"quit":   "Leave without another action.",
}

func askOptions(s snapshot) []string {
	if s.Detached {
		if s.Remotes {
			return []string{"start", "track", "quit"}
		}
		return []string{"start", "quit"}
	}
	if s.Protected {
		return []string{"start", "track", "accept", "list", "quit"}
	}
	if s.HasUpstream && s.Ahead > 0 && s.Behind > 0 {
		return []string{"sync", "push", "accept", "list", "quit"}
	}
	if s.UniqueCommits && (!s.HasUpstream || s.Ahead > 0) {
		return []string{"polish", "push", "accept", "list", "quit"}
	}
	opts := []string{"start", "accept", "list"}
	if s.Remotes {
		opts = append(opts, "track")
	}
	return append(opts, "quit")
}

func askChoices(s snapshot) []prompt.Choice {
	opts := askOptions(s)
	out := make([]prompt.Choice, len(opts))
	for i, action := range opts {
		out[i] = prompt.Choice{Value: action, Description: workActionPurpose[action]}
	}
	return out
}
