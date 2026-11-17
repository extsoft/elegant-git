package work

import (
	"strings"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

var pushID = cmdid.ID{Command: "work", Action: "push"}

func pushSpec(branch *string) argspec.Spec {
	in := argspec.PositionalInput("branch-name", 0, false, "Remote branch name", branch, nil)
	in.OmitInteractive = true
	return argspec.Spec{Inputs: []argspec.Input{in}}
}

func newPushCommand() *cobra.Command {
	var branch string
	spec := pushSpec(&branch)
	c := &cobra.Command{
		Use:   "push [branch-name]",
		Short: "Publishes HEAD to a remote repository",
		Long: `Rebases onto upstream when needed, then force-pushes the current branch to its remote tracking branch or the given branch name.

An in-progress rebase is continued first, including on a protected branch, and conflict resolutions are not stashed or discarded. A new push of a protected branch is refused. When there are uncommitted changes, choose carry, stash, discard, or cancel. carry restores them on this branch afterward and keeps the staged and unstaged split. stash leaves them in the stash instead. discard reverts modifications and deletions of committed files and leaves staged additions and rename targets as untracked files. An ignored file the incoming rebase would write, and for carry an untracked file, is refused before any stash or reset. cancel stops before any of that. A non-interactive run with uncommitted changes fails unless --dirty is set. A clean tree does not need --dirty. A failed fetch, rebase, or push says whether local history changed and whether the remote was updated. Non-interactive mode does not replace Git's rebase editor.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, pushID, func() error {
				return pushRun(cmd, args)
			})
		},
	}
	AddDirtyFlags(c)
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.AttachArgs(c, spec)
	return c
}

func pushRun(cmd *cobra.Command, args []string) error {
	var remoteBranchArg string
	if err := argspec.ResolveCmd(cmd, args, pushSpec(&remoteBranchArg)); err != nil {
		return err
	}
	branch := cliruntime.CurrentBranch()
	return PrepareHistory(History{
		Cmd:  cmd,
		ID:   pushID,
		Verb: "work push",
		Dest: "publishing",
		Allow: func(snap state.Snapshot) bool {
			return snap.Operation == "rebase"
		},
		Protect: func() error {
			if config.IsBranchProtected(branch) {
				cliruntime.ExitProtectedDeliver(branch)
			}
			return nil
		},
		Resolve: func() (string, error) {
			return pushIncoming(branch)
		},
		Replay: true,
		Body: func() error {
			return pushLogic(branch, remoteBranchArg)
		},
	})
}

func pushIncoming(localBranch string) (string, error) {
	if err := git.Verbose("fetch"); err != nil {
		return "", annotate(err, "fetch failed before local history changed")
	}
	source := config.FreshestBranchSourceBranch(localBranch)
	if strings.TrimSpace(source) == "" {
		return "", nil
	}
	return source, nil
}

func pushLogic(localBranch, remoteBranchArg string) error {
	if state.IsThereActiveRebase() {
		if err := continueRebase(); err != nil {
			return annotate(err, "rebase did not finish, and the remote was not updated")
		}
		if !state.IsThereActiveRebase() {
			if err := restoreAside(); err != nil {
				return err
			}
		}
	} else {
		if err := git.Verbose("fetch"); err != nil {
			return annotate(err, "fetch failed before local history changed")
		}
		if err := rebaseOnto(config.FreshestBranchSourceBranch(localBranch)); err != nil {
			return annotate(err, "rebase did not finish, and the remote was not updated")
		}
	}
	upstream := cliruntime.BranchUpstreamShort(localBranch)
	remote := cliruntime.RemoteFromRemoteBranch(upstream)
	branch := pushRemoteBranch(localBranch, remoteBranchArg, upstream)
	if remote == "" {
		remote = config.DefaultUpstreamRemote
	}
	refSpec := localBranch + ":" + branch
	if err := git.VerboseOpLines(cliruntime.OpenURLsIfPossible, "push", "--set-upstream", "--force", remote, refSpec); err != nil {
		return annotate(err, "local rebase finished and the remote push did not")
	}
	return nil
}

func pushRemoteBranch(localBranch, remoteBranchArg, upstream string) string {
	if remoteBranchArg != "" {
		return remoteBranchArg
	}
	if upstream == "" {
		return localBranch
	}
	upstreamBranch := cliruntime.BranchFromRemoteBranch(upstream)
	if upstreamBranch == localBranch {
		return localBranch
	}
	freshestSource := config.FreshestBranchSourceBranch(localBranch)
	sourceBranch := cliruntime.BranchFromRemoteBranch(freshestSource)
	if upstreamBranch == config.DefaultBranch() || upstream == freshestSource || upstreamBranch == sourceBranch {
		return localBranch
	}
	return upstreamBranch
}
