// Package pipe provides stash and branch preservation around command execution.
package pipe

import (
	"fmt"
	"strings"
	"time"

	"github.com/bees-hive/elegant-git/internal/cli/legacy"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/deprecation"
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

// StashPipe runs fn with optional auto-stash around it for id.
func StashPipe(id cmdid.ID, fn func() error) error {
	key := id.ConfigKeySuffix("stash")
	legacyKey := legacyPipeKey(id, "stash")
	_ = gitQuiet("update-index", "-q", "--really-refresh")

	if HasChanges() {
		if msg, _ := localConfigGet(key); msg == "" {
			if msg, _ = localConfigGet(legacyKey); msg != "" {
				deprecation.RecordLegacyPipeKey(legacyKey)
				key = legacyKey
			}
		}
		if msg, _ := localConfigGet(key); msg == "" {
			branch, _ := git.Output("rev-parse", "--abbrev-ref", "HEAD")
			message := fmt.Sprintf("git-elegant %s auto-stash: WIP in '%s' branch on %s",
				id.String(), strings.TrimSpace(branch), time.Now().Format("2006-01-02T15:04:05"))
			if err := git.Verbose("stash", "push", "--message", message); err != nil {
				return err
			}
			if sid := stashID(message); sid != "" {
				_ = localConfigSet(key, message)
			}
		}
	}

	if err := fn(); err != nil {
		return err
	}

	saved, _ := localConfigGet(key)
	if saved == "" {
		saved, _ = localConfigGet(legacyKey)
	}
	if saved != "" {
		_ = gitQuiet("update-index", "-q", "--really-refresh")
		_ = localConfigUnset(key)
		_ = localConfigUnset(legacyKey)
		if sid := stashID(saved); sid != "" {
			return git.Verbose("stash", "pop", sid)
		}
	}
	return nil
}

// BranchPipe restores the original branch after fn if checkout changed it.
func BranchPipe(id cmdid.ID, fn func() error) error {
	key := id.ConfigKeySuffix("current-branch")
	legacyKey := legacyPipeKey(id, "current-branch")
	previous, _ := localConfigGet(key)
	if previous == "" {
		previous, _ = localConfigGet(legacyKey)
		if previous != "" {
			deprecation.RecordLegacyPipeKey(legacyKey)
			key = legacyKey
		}
	}
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
	_ = localConfigUnset(legacyKey)
	now, err := git.Output("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}
	if strings.TrimSpace(now) != previous {
		return git.Verbose("checkout", previous)
	}
	return nil
}

func legacyPipeKey(id cmdid.ID, suffix string) string {
	if legacyName, ok := legacy.IDToLegacy(id); ok {
		return "elegant." + legacyName + "-" + suffix
	}
	return ""
}

func stashID(message string) string {
	out := git.OutputOK("stash", "list", "--grep="+message, "--format=%gd")
	return strings.TrimSpace(strings.Split(out, "\n")[0])
}

func localConfigGet(key string) (string, error) {
	return git.Output("config", "--local", key)
}

func localConfigSet(key, value string) error {
	return git.ConfigLocalSet(key, value)
}

func localConfigUnset(key string) error {
	return git.ConfigLocalUnset(key)
}

func gitQuiet(args ...string) error {
	_, err := git.Output(args...)
	return err
}
