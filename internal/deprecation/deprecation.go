// Package deprecation records deprecated surface usage and prints migrate hints.
package deprecation

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// Event is one deprecation notice for the current process.
type Event struct {
	ID          string
	Surface     string
	Replacement string
	Migrate     string
}

const (
	DEP001 = "DEP-001"
	DEP002 = "DEP-002"
	DEP003 = "DEP-003"
	DEP004 = "DEP-004"
	DEP005 = "DEP-005"
	DEP007 = "DEP-007"
	DEP008 = "DEP-008"
	DEP009 = "DEP-009"
	DEP012 = "DEP-012"

	// SurfaceAnnotation marks a cobra command as a deprecated renamed surface (DEP-012).
	SurfaceAnnotation = "elegant-git.deprecated-surface"
)

var (
	mu     sync.Mutex
	events           = map[string]Event{}
	outW   io.Writer = os.Stderr
)

// SetOutputWriter overrides stderr for tests.
func SetOutputWriter(w io.Writer) {
	outW = w
}

// Record registers a deprecation event once per id per process.
func Record(id, surface, replacement, migrate string) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := events[id]; ok {
		return
	}
	events[id] = Event{ID: id, Surface: surface, Replacement: replacement, Migrate: migrate}
}

// RecordLegacyCommand records DEP-001 for a legacy flat command name.
// replacement is the canonical command path (e.g. "work start").
func RecordLegacyCommand(name, replacement string) {
	if replacement == "" {
		replacement = "see `git elegant --help`"
	}
	Record(DEP001, name, replacement, "git elegant git migrate")
}

// RecordLegacyPersonalHook records DEP-002.
func RecordLegacyPersonalHook(path string) {
	Record(DEP002, "personal hook: "+path, ".git/.config/elegant-git/hooks/<command>-<action>-{ahead,after}", "git elegant repo migrate")
}

// RecordLegacyCommonHook records DEP-003.
func RecordLegacyCommonHook(path string) {
	Record(DEP003, "common hook: "+path, ".config/elegant-git/hooks/<command>-<action>-{ahead,after}", "git elegant hook migrate")
}

// RecordShowCommands records DEP-004.
func RecordShowCommands() {
	Record(DEP004, "command name: show-commands", "git elegant completion <shell>", "git elegant completion bash")
}

// RecordRenamedSurface records DEP-012 for renamed commands or memory keys.
func RecordRenamedSurface(surface, replacement, migrate string) {
	Record(DEP012, surface, replacement, migrate)
}

// Reset clears recorded events (tests).
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	events = map[string]Event{}
}

// Events returns a copy of recorded events.
func Events() []Event {
	mu.Lock()
	defer mu.Unlock()
	out := make([]Event, 0, len(events))
	for _, e := range events {
		out = append(out, e)
	}
	return out
}

// Flush prints one warning line per recorded event to stderr.
func Flush() {
	mu.Lock()
	defer mu.Unlock()
	for _, e := range events {
		fmt.Fprint(outW, formatWarning(e))
	}
}

func formatWarning(e Event) string {
	switch e.ID {
	case DEP001:
		return fmt.Sprintf(
			"Warning: the `%s` command is deprecated; please use `%s`. Run `%s` to migrate automatically.\n",
			e.Surface, e.Replacement, e.Migrate,
		)
	default:
		return fmt.Sprintf("warning: %s is deprecated; use %s. Run `%s` to migrate.\n",
			e.Surface, e.Replacement, e.Migrate)
	}
}
