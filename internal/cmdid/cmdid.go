// Package cmdid defines canonical command identifiers for hooks and config keys.
package cmdid

import "strings"

// ID is {command}.{action}[.{condition}][.{extension}].
type ID struct {
	Command   string
	Action    string
	Condition string
	Extension string
}

// String returns the dotted canonical form (condition and extension omitted when empty).
func (id ID) String() string {
	parts := []string{id.Command, id.Action}
	if id.Condition != "" {
		parts = append(parts, id.Condition)
	}
	if id.Extension != "" {
		parts = append(parts, id.Extension)
	}
	return strings.Join(parts, ".")
}

// HookFileName returns <command>-<action>-<hookType> (e.g. work-save-ahead).
func (id ID) HookFileName(hookType string) string {
	return id.Command + "-" + id.Action + "-" + hookType
}

// ParseDotted parses a canonical dotted id string.
func ParseDotted(s string) (ID, bool) {
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return ID{}, false
	}
	id := ID{Command: parts[0], Action: parts[1]}
	if len(parts) > 2 {
		id.Condition = parts[2]
	}
	if len(parts) > 3 {
		id.Extension = parts[3]
	}
	return id, true
}
