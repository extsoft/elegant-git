package work

import (
	"fmt"
	"strings"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/pipe"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/extsoft/elegant-git/internal/workflows"
	"github.com/spf13/cobra"
)

const acceptWorkBranch = "__eg"

var acceptID = cmdid.ID{Command: "work", Action: "accept"}
var trackIDAccept = cmdid.ID{Command: "work", Action: "track"}

var isAcceptHelperRebase = func() bool {
	return state.IsThereActiveRebase() && state.RebasingBranch() == acceptWorkBranch
}

func newAcceptCommand() *cobra.Command {
	var branch string
	spec := acceptSpec(&branch)
	c := &cobra.Command{
		Use:   "accept <branch>",
		Short: "Adds modifications to the default development branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, acceptID, func() error {
				return acceptRun(cmd, args, spec)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func acceptSpec(branch *string) argspec.Spec {
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInputWithComplete("branch", 0, true, "Branch to accept", branch, nil, sources.BranchNamesUnion, true),
	}}
}

func acceptRun(cmd *cobra.Command, args []string, spec argspec.Spec) error {
	return pipe.StashPipe(acceptID, func() error {
		return pipe.BranchPipe(acceptID, func() error {
			return acceptLogic(cmd, args, spec)
		})
	})
}

func acceptLogic(cmd *cobra.Command, args []string, spec argspec.Spec) error {
	if isAcceptHelperRebase() {
		return git.Verbose("rebase", "--continue")
	}
	if state.IsThereActiveRebase() {
		rb := state.RebasingBranch()
		cliruntime.ExitWorkflowError(fmt.Sprintf("First, please complete current rebase which updates '%s' branch.", rb))
	}
	if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
		return err
	}
	branch := spec.Inputs[0].Get()
	wasLocal := cliruntime.LocalBranchExists(branch)
	if wasLocal {
		if err := git.Verbose("fetch", "--all"); err != nil {
			return err
		}
		if err := git.Verbose("checkout", "-B", acceptWorkBranch, branch); err != nil {
			return err
		}
	} else {
		ctx := cmd.Context()
		workflows.RunAheadCompat(ctx, trackIDAccept, "obtain-work")
		defer workflows.RunAfterCompat(ctx, trackIDAccept, "obtain-work")
		if err := trackExactRemote(branch, acceptWorkBranch); err != nil {
			return err
		}
	}
	if err := git.Verbose("rebase", config.FreshestBranchSourceBranch(branch)); err != nil {
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
			remoteBranch := cliruntime.BranchFromRemoteBranch(actualRemote)
			if err := git.Verbose("push", config.DefaultUpstreamRemote, "--delete", remoteBranch); err != nil {
				return err
			}
		}
	}
	if err := config.ClearBranchSourceBranch(branch); err != nil {
		return err
	}
	if wasLocal && branch != defaultBranch && branch != acceptWorkBranch && !config.IsBranchProtected(branch) {
		if err := git.Verbose("branch", "--delete", "--force", branch); err != nil {
			text.InfoText(fmt.Sprintf("The '%s' branch could not be deleted; it is still present.", branch))
		}
	}
	return nil
}
