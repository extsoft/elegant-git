package work

import (
	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/pipe"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

var syncID = cmdid.ID{Command: "work", Action: "sync"}

func syncSpec(branch *string) argspec.Spec {
	in := argspec.PositionalInput("branch-name", 0, false, "Branch name", branch, nil)
	in.OmitInteractive = true
	return argspec.Spec{Inputs: []argspec.Input{in}}
}

func newSyncCommand() *cobra.Command {
	var branch string
	spec := syncSpec(&branch)
	c := &cobra.Command{
		Use:   "sync [branch-name]",
		Short: "Actualizes the branch with upstream commits",
		Long: `Rebases the current branch onto the freshest source, the branch it was created from. That is not this branch's own pushed upstream.

The branch name is optional and is never asked. Without it, fetches when the repository has remotes, then rebases onto that source: the source branch's upstream if one is set, otherwise origin/<source>, otherwise the local source branch. With a name, rebases onto that branch, fetching first when the name is a remote branch.

If a rebase is already in progress, it is continued first. Uncommitted changes are stashed and restored.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, syncID, func() error {
				return syncRun(cmd, args)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.AttachArgs(c, spec)
	return c
}

func syncRun(cmd *cobra.Command, args []string) error {
	var branch string
	if err := argspec.ResolveCmd(cmd, args, syncSpec(&branch)); err != nil {
		return err
	}
	return pipe.StashPipe(syncID, func() error {
		return syncLogic(branch)
	})
}

func syncLogic(branchArg string) error {
	if state.IsThereActiveRebase() {
		if err := git.Verbose("rebase", "--continue"); err != nil {
			return err
		}
	}
	if branchArg != "" {
		if state.IsRemoteBranch(branchArg) {
			cliruntime.FetchOrInform()
			return git.Verbose("rebase", branchArg)
		}
		return git.Verbose("rebase", branchArg)
	}
	if state.AreThereRemotes() {
		cliruntime.FetchOrInform()
	}
	source := config.FreshestBranchSourceBranch(cliruntime.CurrentBranch())
	return git.Verbose("rebase", source)
}
