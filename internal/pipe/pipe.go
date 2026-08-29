// Package pipe provides stash and branch preservation around command execution.
package pipe

import (
	"fmt"
	"strings"
	"time"

	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/git"
	cmdmem "github.com/extsoft/elegant-git/internal/memory/cmd"
)

// HasChanges reports whether HEAD has staged or unstaged changes.
func HasChanges() bool {
	if err := gitQuiet("diff-index", "--quiet", "HEAD"); err == nil {
		if err := gitQuiet("diff-index", "--cached", "--quiet", "HEAD"); err == nil {
			return false
		}
	}
	return true
}

// StashPipe runs fn with optional auto-stash around it for id.
func StashPipe(id cmdid.ID, fn func() error) error {
	_ = gitQuiet("update-index", "-q", "--really-refresh")

	if HasChanges() {
		if msg, _ := cmdmem.Get(id, cmdmem.FieldStash); msg == "" {
			branch, _ := git.Output("rev-parse", "--abbrev-ref", "HEAD")
			message := fmt.Sprintf("eg %s auto-stash: WIP in '%s' branch on %s",
				id.String(), strings.TrimSpace(branch), time.Now().Format("2006-01-02T15:04:05"))
			if err := git.Verbose("stash", "push", "--message", message); err != nil {
				return err
			}
			if sid := stashID(message); sid != "" {
				_ = cmdmem.Set(id, cmdmem.FieldStash, message)
			}
		}
	}

	if err := fn(); err != nil {
		return err
	}

	saved, _ := cmdmem.Get(id, cmdmem.FieldStash)
	if saved != "" {
		_ = gitQuiet("update-index", "-q", "--really-refresh")
		_ = cmdmem.Unset(id, cmdmem.FieldStash)
		if sid := stashID(saved); sid != "" {
			return git.Verbose("stash", "pop", sid)
		}
	}
	return nil
}

// BranchPipe restores the original branch after fn if checkout changed it.
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
	if strings.TrimSpace(now) != previous {
		return git.Verbose("checkout", previous)
	}
	return nil
}

func stashID(message string) string {
	out := git.OutputOK("stash", "list", "--grep="+message, "--format=%gd")
	return strings.TrimSpace(strings.Split(out, "\n")[0])
}

func gitQuiet(args ...string) error {
	_, err := git.Output(args...)
	return err
}
