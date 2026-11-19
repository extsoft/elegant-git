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
		Long: `Adds modifications to the default development branch.

A rebase already in progress on the helper branch __eg is continued first and, when that rebase finishes, the merge and push follow. The name you pass has to be the branch that rebase started from; a different name is refused before the rebase is continued. With no name, the one local branch still pointing at that starting commit is used. The remote branch is deleted only when its name is that accepted branch, it is not the default branch, and it is not protected. Conflict resolutions are not stashed or discarded. Any other in-progress operation is refused before uncommitted changes are touched. Otherwise, when there are uncommitted changes, choose carry, stash, discard, or cancel. carry restores them on the branch you started on, when that branch still exists, and keeps the staged and unstaged split. If that branch was deleted, the backup stays and is not applied to another branch. stash leaves them in the stash instead. discard reverts modifications and deletions of committed files and leaves staged additions and rename targets as untracked files. An ignored file the incoming rebase would write, and for carry an untracked file, is refused before any stash or reset. cancel stops before any of that. A non-interactive run with uncommitted changes fails unless --dirty is set. A clean tree does not need --dirty. A failed fetch, rebase, or push says whether the local merge finished and whether the remote was updated. Re-running after an external continue or abort does not push or delete again. Accept will not replace __eg when that helper already has commits that are not on the accepted branch.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, acceptID, func() error {
				return acceptRun(cmd, args, spec)
			})
		},
	}
	AddDirtyFlags(c)
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func acceptSpec(branch *string) argspec.Spec {
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.GitBacked(argspec.PositionalInputWithComplete("branch", 0, true, "Branch to accept", branch, nil, sources.BranchNamesUnion, true)),
	}}
}

func acceptRun(cmd *cobra.Command, args []string, spec argspec.Spec) error {
	return PrepareHistory(History{
		Cmd:  cmd,
		ID:   acceptID,
		Verb: "work accept",
		Dest: "returning to the original branch",
		Allow: func(state.Snapshot) bool {
			return isAcceptHelperRebase()
		},
		Mismatch: func(snap state.Snapshot) error {
			if snap.Operation == "rebase" {
				return workflowText(fmt.Sprintf("First, please complete current rebase which updates '%s' branch.", state.RebasingBranch()))
			}
			return operationError(snap.Operation)
		},
		Resolve: func() (string, error) {
			return acceptIncoming(cmd, args, spec)
		},
		Replay: true,
		Footprint: func(source string) ([]string, error) {
			return acceptFootprint(spec.Inputs[0].Get(), source)
		},
		Body: func() error {
			var done bool
			err := pipe.BranchPipe(acceptID, func() error {
				var err error
				done, err = acceptWork(cmd, args, spec)
				return err
			})
			if err != nil {
				return err
			}
			if !done {
				return nil
			}
			return restoreAside()
		},
	})
}

func acceptBranchForRebase(cmd *cobra.Command, args []string, spec argspec.Spec, orig string) (string, error) {
	orig = strings.TrimSpace(orig)
	if len(args) == 0 {
		return localBranchAt(orig), nil
	}
	if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
		return "", err
	}
	branch := spec.Inputs[0].Get()
	tip, err := git.Output("rev-parse", "--verify", branch+"^{commit}")
	if err != nil || strings.TrimSpace(tip) != orig {
		started := localBranchAt(orig)
		if started == "" {
			started = orig
		}
		return "", workflowText(fmt.Sprintf("This %s rebase started from '%s'. Accept will not continue it as '%s'.", acceptWorkBranch, started, branch))
	}
	return branch, nil
}

func localBranchAt(oid string) string {
	oid = strings.TrimSpace(oid)
	if oid == "" {
		return ""
	}
	out := git.OutputOK("for-each-ref", "--format=%(objectname) %(refname:short)", "refs/heads")
	found := ""
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) != 2 || fields[0] != oid || fields[1] == acceptWorkBranch {
			continue
		}
		if found != "" {
			return ""
		}
		found = fields[1]
	}
	return found
}

func acceptRemoteDelete(branch, upstream string) (string, bool) {
	if !strings.HasPrefix(upstream, config.DefaultUpstreamRemote+"/") {
		return "", false
	}
	remoteBranch := cliruntime.BranchFromRemoteBranch(upstream)
	accepted := branch
	if strings.HasPrefix(accepted, config.DefaultUpstreamRemote+"/") {
		accepted = cliruntime.BranchFromRemoteBranch(accepted)
	}
	if remoteBranch == "" || remoteBranch != accepted || remoteBranch == config.DefaultBranch() || config.IsBranchProtected(remoteBranch) {
		return "", false
	}
	return remoteBranch, true
}

func refuseHelperRebuild(source string) error {
	source = strings.TrimSpace(source)
	if source == "" || source == acceptWorkBranch || !cliruntime.LocalBranchExists(acceptWorkBranch) {
		return nil
	}
	out, err := git.Output("rev-list", "--count", source+".."+acceptWorkBranch)
	n := strings.TrimSpace(out)
	if err != nil || (n != "" && n != "0") {
		return workflowText(fmt.Sprintf("The helper branch %s has commits that are not on '%s'. Accept will not replace it. Merge %s into the default branch, or delete %s to start accept again.", acceptWorkBranch, source, acceptWorkBranch, acceptWorkBranch))
	}
	return nil
}

func acceptLogic(cmd *cobra.Command, args []string, spec argspec.Spec) error {
	_, err := acceptWork(cmd, args, spec)
	return err
}

func acceptIncoming(cmd *cobra.Command, args []string, spec argspec.Spec) (string, error) {
	if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
		return "", err
	}
	branch := spec.Inputs[0].Get()
	if err := refuseHelperRebuild(branch); err != nil {
		return "", err
	}
	if state.AreThereRemotes() {
		if err := git.Verbose("fetch", "--all"); err != nil {
			return "", annotate(err, "fetch failed before local history changed")
		}
	}
	source := config.FreshestBranchSourceBranch(branch)
	if strings.TrimSpace(source) == "" {
		return "", nil
	}
	return source, nil
}

func acceptFootprint(branch, source string) ([]string, error) {
	branch = strings.TrimSpace(branch)
	source = strings.TrimSpace(source)
	if branch == "" {
		return diffPaths("HEAD", source)
	}
	var writes []string
	for _, pair := range [][2]string{{"HEAD", branch}, {branch, source}} {
		paths, err := diffPaths(pair[0], pair[1])
		if err != nil {
			return nil, err
		}
		writes = append(writes, paths...)
	}
	adds, err := replayAdds(source, branch)
	if err != nil {
		return nil, err
	}
	return append(writes, adds...), nil
}

func acceptWork(cmd *cobra.Command, args []string, spec argspec.Spec) (bool, error) {
	continuing := isAcceptHelperRebase()
	branch := ""
	if continuing {
		marker, ok := state.ActiveRebase()
		if !ok {
			if len(args) > 0 {
				return false, workflowText(fmt.Sprintf("This %s rebase cannot be identified, so accept will not continue it.", acceptWorkBranch))
			}
		} else {
			var err error
			branch, err = acceptBranchForRebase(cmd, args, spec, marker.Orig)
			if err != nil {
				return false, err
			}
		}
		if err := continueRebase(); err != nil {
			return false, annotate(err, "rebase did not finish, and the remote was not updated")
		}
		if state.IsThereActiveRebase() {
			return false, nil
		}
	} else if state.IsThereActiveRebase() {
		rb := state.RebasingBranch()
		cliruntime.ExitWorkflowError(fmt.Sprintf("First, please complete current rebase which updates '%s' branch.", rb))
	}
	if branch == "" && (!continuing || len(args) > 0) {
		if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
			return false, err
		}
		branch = spec.Inputs[0].Get()
	}
	wasLocal := branch != "" && cliruntime.LocalBranchExists(branch)
	upstream := ""
	if wasLocal {
		upstream = strings.TrimSpace(git.OutputOK("for-each-ref", "--format=%(upstream:short)", "refs/heads/"+branch))
	}
	if !continuing {
		if err := refuseHelperRebuild(branch); err != nil {
			return false, err
		}
		if wasLocal {
			if err := git.Verbose("fetch", "--all"); err != nil {
				return false, annotate(err, "fetch failed before local history changed")
			}
			if err := git.Verbose("checkout", "-B", acceptWorkBranch, branch); err != nil {
				return false, err
			}
		} else {
			ctx := cmd.Context()
			workflows.RunAheadCompat(ctx, trackIDAccept, "obtain-work")
			defer workflows.RunAfterCompat(ctx, trackIDAccept, "obtain-work")
			if err := trackExactRemote(branch, acceptWorkBranch); err != nil {
				return false, err
			}
		}
		if err := rebaseHoldingAside(config.FreshestBranchSourceBranch(branch)); err != nil {
			return false, annotate(err, "rebase did not finish, and the remote was not updated")
		}
	}
	if upstream == "" {
		upstream = strings.TrimSpace(git.OutputOK("for-each-ref", "--format=%(upstream:short)", "refs/heads/"+acceptWorkBranch))
	}
	defaultBranch := config.DefaultBranch()
	if err := git.Verbose("checkout", defaultBranch); err != nil {
		return false, err
	}
	if err := git.Verbose("merge", "--ff-only", acceptWorkBranch); err != nil {
		return false, err
	}
	if err := git.Verbose("branch", "--delete", "--force", acceptWorkBranch); err != nil {
		return false, err
	}
	if state.AreThereRemotes() {
		if err := git.Verbose("push", config.DefaultUpstreamRemote, defaultBranch+":"+defaultBranch); err != nil {
			return false, annotate(err, "local merge finished and the remote push did not")
		}
		if remoteBranch, ok := acceptRemoteDelete(branch, upstream); ok {
			if err := git.Verbose("push", config.DefaultUpstreamRemote, "--delete", remoteBranch); err != nil {
				return false, annotate(err, "local merge and the default-branch push finished, and deleting the remote branch did not")
			}
			text.InfoText(fmt.Sprintf("Deleted the remote branch '%s'.", remoteBranch))
		}
	}
	if branch != "" {
		if err := config.ClearBranchSourceBranch(branch); err != nil {
			return false, err
		}
	}
	if wasLocal && branch != defaultBranch && branch != acceptWorkBranch && !config.IsBranchProtected(branch) {
		if err := git.Verbose("branch", "--delete", "--force", branch); err != nil {
			text.InfoText(fmt.Sprintf("The '%s' branch could not be deleted; it is still present.", branch))
		}
	}
	return true, nil
}
