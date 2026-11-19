package release

import (
	"fmt"
	"os"
	"strings"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/work"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/pipe"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

var newID = cmdid.ID{Command: "release", Action: "new"}

func releaseNewSpec(name *string) argspec.Spec {
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInput("name", 0, true, "Release tag name", name, nil),
	}}
}

func newNewCommand() *cobra.Command {
	var tagName string
	spec := releaseNewSpec(&tagName)
	c := &cobra.Command{
		Use:   "new <name>",
		Short: "Releases the default development branch",
		Long: `Checks out the default branch, tags a new release, pushes tags, and prints release notes.

When there are uncommitted changes, choose carry, stash, discard, or cancel. carry restores them on the branch you started on, when that branch still exists, and keeps the staged and unstaged split. If that branch is gone, the backup stays and is not applied to another branch. stash leaves them in the stash instead. discard restores modifications and deletions of committed files. Without --untracked=delete it leaves staged additions, rename targets, and other non-ignored untracked files in place. --untracked=delete removes those listed new-path files permanently and keeps no backup. An ignored file the incoming pull would write, and for carry an untracked file, is refused before any stash or reset. cancel stops before any of that. A non-interactive run with uncommitted changes fails unless --dirty is set. A clean tree does not need --dirty. An in-progress Git operation is refused before that choice. The tag message is written outside the worktree. A failed pull or tag push says whether the local tag exists. Re-running after an external continue or abort does not push or tag again. Non-interactive mode does not replace the tag editor.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, newID, func() error {
				return newRun(cmd, args)
			})
		},
	}
	work.AddDirtyFlags(c)
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.AttachArgs(c, spec)
	return c
}

func newRun(cmd *cobra.Command, args []string) error {
	return work.PrepareHistory(work.History{
		Cmd:     cmd,
		ID:      newID,
		Verb:    "release new",
		Dest:    "returning to the original branch",
		Resolve: releaseIncoming,
		Body: func() error {
			return pipe.BranchPipe(newID, func() error {
				return newLogic(cmd, args)
			})
		},
	})
}

func releaseIncoming() (string, error) {
	if !state.AreThereRemotes() {
		return config.DefaultBranch(), nil
	}
	if err := git.Verbose("fetch"); err != nil {
		return "", fmt.Errorf("fetch failed before local history changed: %w", err)
	}
	remote := config.DefaultUpstreamRemote + "/" + config.DefaultBranch()
	if _, err := git.Output("rev-parse", "--verify", "--quiet", remote+"^{commit}"); err != nil {
		return "", nil
	}
	return remote, nil
}

func newLogic(cmd *cobra.Command, args []string) error {
	var tagName string
	if err := argspec.ResolveCmd(cmd, args, releaseNewSpec(&tagName)); err != nil {
		return err
	}
	defaultBranch := config.DefaultBranch()
	if err := git.Verbose("checkout", defaultBranch); err != nil {
		return err
	}
	if err := git.Verbose("pull", "--tags"); err != nil {
		return fmt.Errorf("pull failed before the tag was created: %w", err)
	}
	lastTag := state.LastTag()
	newTag := tagName
	diapason := lastTag + "...HEAD"
	notesFrom := lastTag
	if lastTag == "" {
		diapason = "HEAD"
		notesFrom = "all-commits"
	}
	var body strings.Builder
	fmt.Fprintf(&body, "Release %s\n\n", newTag)
	logOut := git.OutputOK("log", diapason, "--pretty=format:- %s", "--reverse")
	if logOut != "" {
		body.WriteString(logOut)
		body.WriteString("\n")
	}
	body.WriteString("\n\n")
	messageFile, err := os.CreateTemp("", "eg-tag-message-*")
	if err != nil {
		return err
	}
	messagePath := messageFile.Name()
	defer os.Remove(messagePath)
	if _, err := messageFile.WriteString(body.String()); err != nil {
		_ = messageFile.Close()
		return err
	}
	if err := messageFile.Close(); err != nil {
		return err
	}
	if err := git.Verbose("tag", "--annotate", "--file", messagePath, "--edit", newTag); err != nil {
		return err
	}
	if err := git.Verbose("push", "--tags"); err != nil {
		return fmt.Errorf("the tag %s was created locally and the remote tag push did not: %w", newTag, err)
	}
	notes, err := formatReleaseNotes([]string{"smart", notesFrom, "HEAD"})
	if err != nil {
		return err
	}
	cliruntime.CopyNotesIfPossible(notes)
	return nil
}
