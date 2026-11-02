package work

import (
	"strings"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/pipe"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

var syncID = cmdid.ID{Command: "work", Action: "sync"}

func syncSpec(branch *string) argspec.Spec {
	in := argspec.PositionalInputWithComplete("branch-name", 0, false, "Branch name", branch, nil, syncBranchComplete, false)
	in.PromptOptional = true
	return argspec.Spec{Inputs: []argspec.Input{in}}
}

func newSyncCommand() *cobra.Command {
	var branch string
	spec := syncSpec(&branch)
	c := &cobra.Command{
		Use:   "sync [branch-name]",
		Short: "Rebases onto the source branch, or a named branch",
		Long: `Rebases the current branch onto the freshest source, the branch it was created from. That is not this branch's own pushed upstream.

The branch name is optional. In interactive mode, fetches once when the repository has remotes, then asks which branch to rebase onto. The picker lists the source branch first (the default selection), the default development branch, and this branch's upstream when those refs exist, then every other local and remote branch. The current branch is left out of that list; its upstream can still appear as a pinned row. It does not fetch again after you choose. With a name on the command line, rebases onto that branch, fetching once first when the name is a remote branch. In non-interactive mode, an omitted name rebases onto the source branch without asking.

If a rebase is already in progress, it is continued first. Uncommitted changes are stashed and restored.

Commits whose changes are already in that branch are left out of the rebase. A source branch already in the default development branch, or whose upstream is gone, counts as delivered, so the rebase uses the default development branch. When the local source branch differs from its remote-tracking branch and already contains it, the local branch is used.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, syncID, func() error {
				return syncRun(cmd, args)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func syncRun(cmd *cobra.Command, args []string) error {
	var branch string
	if err := argspec.ResolveCmd(cmd, args, syncSpec(&branch)); err != nil {
		return err
	}
	fetchedForPicker := len(args) == 0 && strings.TrimSpace(branch) != ""
	return pipe.StashPipe(syncID, func() error {
		return syncLogic(branch, fetchedForPicker)
	})
}

func syncLogic(branchArg string, fetchedForPicker bool) error {
	if state.IsThereActiveRebase() {
		if err := continueRebase(); err != nil {
			return err
		}
	}
	if branchArg != "" {
		if state.IsRemoteBranch(branchArg) && !fetchedForPicker {
			cliruntime.FetchOrInform()
		}
		return rebaseOnto(branchArg)
	}
	if state.AreThereRemotes() {
		cliruntime.FetchOrInform()
	}
	source := config.FreshestBranchSourceBranch(cliruntime.CurrentBranch())
	return rebaseOnto(source)
}
