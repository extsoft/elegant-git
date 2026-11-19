package git

import (
	"fmt"
	"strings"
)

// BeforeDropStash runs after the selector is chosen and before git stash drop.
// Tests use it to check that the drop uses the selector chosen before this hook.
var BeforeDropStash func()

// DropStash resolves one stash entry to its current selector and drops it.
// oid is the full commit. message, when set, must equal the subject after Git's
// "On <branch>: " prefix. No match means the entry is already gone. The same
// commit under another message is left in place. Callers hold any recovery
// lock; this function does not take one.
func DropStash(oid, message string) error {
	if oid == "" {
		return fmt.Errorf("cannot drop a stash without a commit")
	}
	out, err := Output("stash", "list", "--format=%H%x00%gd%x00%gs")
	if err != nil {
		return err
	}
	var selector string
	sawOID := false
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x00")
		if len(parts) < 3 || parts[0] == "" || !validStashSelector(parts[1]) {
			return fmt.Errorf("stash list entry %q cannot be identified", line)
		}
		if parts[0] != oid {
			continue
		}
		sawOID = true
		if message != "" && stashSubjectMessage(strings.Join(parts[2:], "\x00")) != message {
			continue
		}
		if selector != "" {
			return fmt.Errorf("stash %s is ambiguous; other stashes were not changed", oid)
		}
		selector = parts[1]
	}
	if selector == "" {
		if sawOID && message != "" {
			return fmt.Errorf("stash %s changed identity; other stashes were not changed", oid)
		}
		return nil
	}
	if BeforeDropStash != nil {
		BeforeDropStash()
	}
	return Verbose("stash", "drop", selector)
}

func validStashSelector(ref string) bool {
	rest, ok := strings.CutPrefix(ref, "stash@{")
	if !ok {
		return false
	}
	n, ok := strings.CutSuffix(rest, "}")
	if !ok || n == "" {
		return false
	}
	for _, c := range n {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func stashSubjectMessage(subject string) string {
	rest, ok := strings.CutPrefix(subject, "On ")
	if !ok {
		return subject
	}
	_, msg, ok := strings.Cut(rest, ": ")
	if !ok {
		return subject
	}
	return msg
}
