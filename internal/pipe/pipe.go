// Package pipe provides stash and branch preservation around command execution.
package pipe

import (
	"fmt"
	"strings"
	"time"

	"github.com/bees-hive/elegant-git/internal/git"
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

// StashPipe runs fn with optional auto-stash around it for command.
func StashPipe(command string, fn func() error) error {
	key := fmt.Sprintf("elegant.%s-stash", command)
	_ = gitQuiet("update-index", "-q", "--really-refresh")

	if HasChanges() {
		if msg, _ := localConfigGet(key); msg == "" {
			branch, _ := git.Output("rev-parse", "--abbrev-ref", "HEAD")
			message := fmt.Sprintf("git-elegant %s auto-stash: WIP in '%s' branch on %s",
				command, strings.TrimSpace(branch), time.Now().Format("2006-01-02T15:04:05"))
			if err := git.Verbose("stash", "push", "--message", message); err != nil {
				return err
			}
			if id := stashID(message); id != "" {
				_ = localConfigSet(key, message)
			}
		}
	}

	if err := fn(); err != nil {
		return err
	}

	saved, _ := localConfigGet(key)
	if saved != "" {
		_ = gitQuiet("update-index", "-q", "--really-refresh")
		_ = localConfigUnset(key)
		if id := stashID(saved); id != "" {
			return git.Verbose("stash", "pop", id)
		}
	}
	return nil
}

// BranchPipe restores the original branch after fn if checkout changed it.
func BranchPipe(command string, fn func() error) error {
	key := fmt.Sprintf("elegant.%s-current-branch", command)
	previous, _ := localConfigGet(key)
	if previous == "" {
		cur, err := git.Output("rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return err
		}
		previous = strings.TrimSpace(cur)
		_ = localConfigSet(key, previous)
	}

	if err := fn(); err != nil {
		return err
	}

	_ = localConfigUnset(key)
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

func localConfigGet(key string) (string, error) {
	return git.Output("config", "--local", key)
}

func localConfigSet(key, value string) error {
	return git.Verbose("config", "--local", key, value)
}

func localConfigUnset(key string) error {
	_, err := git.Output("config", "--local", "--unset", key)
	return err
}

func gitQuiet(args ...string) error {
	_, err := git.Output(args...)
	return err
}
