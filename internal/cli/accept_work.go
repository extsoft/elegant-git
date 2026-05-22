package cli

import (
	"fmt"
	"strings"

	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/pipe"
	"github.com/bees-hive/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

const acceptWorkBranch = "__eg"

func newAcceptWorkCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithWorkflows(cmd, func() error {
				return acceptWorkRun(args)
			})
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func acceptWorkRun(args []string) error {
	return pipe.StashPipe("accept-work", func() error {
		return pipe.BranchPipe("accept-work", func() error {
			return acceptWorkLogic(args)
		})
	})
}

func acceptWorkLogic(args []string) error {
	if state.IsThereActiveRebase() {
		rb := state.RebasingBranch()
		if rb == acceptWorkBranch {
			return git.Verbose("rebase", "--continue")
		}
		exitWorkflowError(fmt.Sprintf("First, please complete current rebase which updates '%s' branch.", rb))
	}
	requireArgs(args, "Please provide a branch name.")
	changes := args[0]
	if localBranchExists(changes) {
		if err := git.Verbose("fetch", "--all"); err != nil {
			return err
		}
		if err := git.Verbose("checkout", "-B", acceptWorkBranch, changes); err != nil {
			return err
		}
	} else {
		if err := runObtainWorkHooks(func() error {
			return obtainWorkLogic(changes, acceptWorkBranch)
		}); err != nil {
			return err
		}
	}
	if err := git.Verbose("rebase", config.FreshestDefaultBranch()); err != nil {
		return err
	}
	actualRemote := git.OutputOK("for-each-ref", "--format=%(upstream:short)", "refs/heads/"+acceptWorkBranch)
	defaultBranch := config.DefaultBranch()
	if err := git.Verbose("checkout", defaultBranch); err != nil {
		return err
	}
	if err := git.Verbose("merge", "--ff-only", acceptWorkBranch); err != nil {
		return err
	}
	if err := git.Verbose("branch", "--delete", "--force", acceptWorkBranch); err != nil {
		return err
	}
	if state.AreThereRemotes() {
		if err := git.Verbose("push", config.DefaultUpstreamRemote, defaultBranch+":"+defaultBranch); err != nil {
			return err
		}
		if strings.HasPrefix(actualRemote, config.DefaultUpstreamRemote+"/") {
			remoteBranch := branchFromRemoteBranch(actualRemote)
			if err := git.Verbose("push", config.DefaultUpstreamRemote, "--delete", remoteBranch); err != nil {
				return err
			}
		}
	}
	return nil
}
