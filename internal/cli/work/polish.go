package work

import (
	"fmt"
	"strings"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var polishID = cmdid.ID{Command: "work", Action: "polish"}

func newPolishCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "polish",
		Short: "Rebases HEAD interactively",
		Long: `Rebases the commits you added onto the branch this one was created from.

An in-progress rebase is continued first, including on a protected branch, and conflict resolutions are not stashed or discarded. A new rebase on a protected branch is refused. When there are uncommitted changes, choose carry, stash, discard, or cancel. carry restores them on this branch afterward and keeps the staged and unstaged split. stash leaves them in the stash instead. discard reverts modifications and deletions of committed files and leaves staged additions and rename targets as untracked files. An ignored file the incoming rebase would write, and for carry an untracked file, is refused before any stash or reset. cancel stops before any of that. A non-interactive run with uncommitted changes fails unless --dirty is set. A clean tree does not need --dirty. Non-interactive mode does not replace Git's rebase editor.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cliruntime.RunWithWorkflows(cmd, polishID, func() error {
				return polishRun(cmd)
			})
		},
	}
	AddDirtyFlags(c)
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func polishRun(cmd *cobra.Command) error {
	return PrepareHistory(History{
		Cmd:  cmd,
		ID:   polishID,
		Verb: "work polish",
		Dest: "the rebase",
		Allow: func(snap state.Snapshot) bool {
			return snap.Operation == "rebase"
		},
		Protect: func() error {
			branch := cliruntime.CurrentBranch()
			if config.IsBranchProtected(branch) {
				cliruntime.ExitProtectedNoRewrite(branch)
			}
			return nil
		},
		Resolve: polishIncoming,
		Replay:  true,
		Body:    polishBody,
	})
}

func polishIncoming() (string, error) {
	latest := config.FreshestBranchSourceBranch(cliruntime.CurrentBranch())
	if strings.TrimSpace(latest) == "" {
		return "", nil
	}
	commits := strings.Fields(git.OutputOK("rev-list", latest+"..HEAD"))
	if len(commits) == 0 {
		return "HEAD", nil
	}
	return fmt.Sprintf("HEAD~%d", len(commits)), nil
}

func polishBody() error {
	if state.IsThereActiveRebase() {
		if err := continueRebase(); err != nil {
			return err
		}
		if state.IsThereActiveRebase() {
			return nil
		}
		return restoreAside()
	}
	branch := cliruntime.CurrentBranch()
	latest := config.FreshestBranchSourceBranch(branch)
	commits := strings.Fields(git.OutputOK("rev-list", latest+"..@"))
	if len(commits) == 0 {
		text.InfoText(fmt.Sprintf("There are no new commits comparing to '%s' branch.", latest))
		return restoreAside()
	}
	n := len(commits)
	base := fmt.Sprintf("@~%d", n)
	return asideAround(base, base, true, func() error {
		return git.Verbose("rebase", "--interactive", base)
	})
}
