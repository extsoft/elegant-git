// Package cmd implements per-command transient state inside .git/elegant-git/.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/extsoft/elegant-git/internal/cmdid"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
)

const (
	dirName  = "elegant-git"
	fileName = "commands.json"

	// FieldStash is the pipe stash message field on an entry.
	FieldStash = "stash"
	// FieldBranch is the pipe preserved branch field on an entry.
	FieldBranch = "branch"
)

// Entry holds transient state for one command invocation.
type Entry struct {
	Stash  string `json:"stash,omitempty"`
	Branch string `json:"branch,omitempty"`
}

// State maps canonical command id (e.g. work.accept) to entry.
type State map[string]*Entry

// Path returns the commands state file path for a git directory.
func Path(gitDir string) string {
	if override := os.Getenv("ELEGANT_GIT_CMD_STATE_FILE"); override != "" {
		return override
	}
	return filepath.Join(gitDir, dirName, fileName)
}

// Load reads command state from gitDir.
func Load(gitDir string) (State, error) {
	p := Path(gitDir)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s == nil {
		return State{}, nil
	}
	return s, nil
}

// Save atomically writes command state (creates parent dir; keeps file even when empty).
func Save(gitDir string, s State) error {
	if s == nil {
		s = State{}
	}
	p := Path(gitDir)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// Get returns a field value for id in the current repository.
func Get(id cmdid.ID, field string) (string, error) {
	gitDir, err := memrepo.GitDir()
	if err != nil {
		return "", err
	}
	s, err := Load(gitDir)
	if err != nil {
		return "", err
	}
	return fieldValue(s[id.String()], field), nil
}

// Set sets a field value for id in the current repository.
func Set(id cmdid.ID, field, value string) error {
	gitDir, err := memrepo.GitDir()
	if err != nil {
		return err
	}
	s, err := Load(gitDir)
	if err != nil {
		return err
	}
	key := id.String()
	e := s[key]
	if e == nil {
		e = &Entry{}
		s[key] = e
	}
	if err := setField(e, field, value); err != nil {
		return err
	}
	return Save(gitDir, s)
}

// Unset clears a field for id in the current repository.
func Unset(id cmdid.ID, field string) error {
	gitDir, err := memrepo.GitDir()
	if err != nil {
		return err
	}
	s, err := Load(gitDir)
	if err != nil {
		return err
	}
	key := id.String()
	e := s[key]
	if e == nil {
		return Save(gitDir, s)
	}
	if err := setField(e, field, ""); err != nil {
		return err
	}
	if entryEmpty(e) {
		delete(s, key)
	}
	return Save(gitDir, s)
}

func fieldValue(e *Entry, field string) string {
	if e == nil {
		return ""
	}
	switch field {
	case FieldStash:
		return e.Stash
	case FieldBranch:
		return e.Branch
	default:
		return ""
	}
}

func setField(e *Entry, field, value string) error {
	switch field {
	case FieldStash:
		e.Stash = value
	case FieldBranch:
		e.Branch = value
	default:
		return fmt.Errorf("cmd state: unknown field %q", field)
	}
	return nil
}

func entryEmpty(e *Entry) bool {
	return e == nil || (e.Stash == "" && e.Branch == "")
}
