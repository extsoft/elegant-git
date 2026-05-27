package work

import (
	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/pipe"
	"github.com/bees-hive/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

var syncID = cmdid.ID{Command: "work", Action: "sync"}

func newSyncCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "sync [branch-name]",
		Short: "Actualizes the branch with upstream commits",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, syncID, func() error {
				return syncRun(cmd, args)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func syncRun(cmd *cobra.Command, args []string) error {
	var branch string
	if err := argspec.ResolveCmd(cmd, args, argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInput("branch-name", 0, false, "Branch name", &branch, nil),
	}}); err != nil {
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
		return git.Verbose("rebase", config.DefaultRemoteTrackingBranch())
	}
	return git.Verbose("rebase", config.DefaultBranch())
}
