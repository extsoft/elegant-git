// Package pipe provides stash and branch preservation around command execution.
package pipe

import (
	"fmt"
	"strings"

	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/git"
	cmdmem "github.com/extsoft/elegant-git/internal/memory/cmd"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
)

// HasChanges reports whether HEAD has staged or unstaged changes.
func HasChanges() bool {
	_ = gitQuiet("update-index", "-q", "--refresh")
	if err := gitQuiet("diff-index", "--quiet", "HEAD"); err == nil {
		if err := gitQuiet("diff-index", "--cached", "--quiet", "HEAD"); err == nil {
			return false
		}
	}
	return true
}

// StashPipe runs fn with optional auto-stash around it for id.
// A failed fn leaves the stash in place so the command can resume.
func StashPipe(id cmdid.ID, fn func() error) error {
	return runStashPipe(id, fn, false)
}

// StashPipeRestoring runs fn like StashPipe and restores the stash when fn fails.
func StashPipeRestoring(id cmdid.ID, fn func() error) error {
	return runStashPipe(id, fn, true)
}

// BranchPipe restores the original branch after fn if checkout changed it and
// that local branch still exists.
func BranchPipe(id cmdid.ID, fn func() error) error {
	previous, _ := cmdmem.Get(id, cmdmem.FieldBranch)
	if previous == "" {
		cur, err := git.Output("rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return err
		}
		previous = strings.TrimSpace(cur)
		_ = cmdmem.Set(id, cmdmem.FieldBranch, previous)
	}

	if err := fn(); err != nil {
		return err
	}

	_ = cmdmem.Unset(id, cmdmem.FieldBranch)
	now, err := git.Output("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}
	if strings.TrimSpace(now) == previous {
		return nil
	}
	if !state.LocalBranchExists(previous) {
		text.InfoText(fmt.Sprintf("The '%s' branch no longer exists; staying on '%s'.", previous, strings.TrimSpace(now)))
		return nil
	}
	return git.Verbose("checkout", previous)
}

func stashWIPBranch(message string) string {
	const mark = " auto-stash: WIP in '"
	i := strings.Index(message, mark)
	if i < 0 {
		return ""
	}
	rest := message[i+len(mark):]
	j := strings.LastIndex(rest, "' branch on ")
	if j < 0 {
		return ""
	}
	return rest[:j]
}

func gitQuiet(args ...string) error {
	_, err := git.Output(args...)
	return err
}
