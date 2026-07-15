// Package repo implements per-repository elegant-git state inside .git/elegant-git/.
package repo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bees-hive/elegant-git/internal/git"
)

const (
	SchemaVersion = 1
	dirName       = "elegant-git"
	fileName      = "state.json"
)

// State is the per-repo memory document.
type State struct {
	SchemaVersion     int               `json:"schema_version"`
	RepoID            string            `json:"repo_id"`
	ProfileID         string            `json:"profile_id"`
	DefaultBranch     string            `json:"default_branch"`
	ProtectedBranches []string          `json:"protected_branches"`
	BranchSources     map[string]string `json:"branch_sources,omitempty"`
}

// Path returns the state file path for a git directory.
func Path(gitDir string) string {
	if override := os.Getenv("ELEGANT_GIT_REPO_STATE_FILE"); override != "" {
		return override
	}
	return filepath.Join(gitDir, dirName, fileName)
}

// Load reads per-repo state from gitDir.
func Load(gitDir string) (*State, error) {
	p := Path(gitDir)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{SchemaVersion: SchemaVersion}, nil
		}
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.SchemaVersion == 0 {
		s.SchemaVersion = SchemaVersion
	}
	return &s, nil
}

// Save atomically writes per-repo state.
func Save(gitDir string, s *State) error {
	if s.SchemaVersion == 0 {
		s.SchemaVersion = SchemaVersion
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

// GitDir returns the absolute .git directory for the current repository.
func GitDir() (string, error) {
	out := git.OutputOK("rev-parse", "--git-dir")
	if out == "" {
		return "", fmt.Errorf("not a git repository")
	}
	if !filepath.IsAbs(out) {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		out = filepath.Join(wd, out)
	}
	return filepath.Clean(out), nil
}

// LoadFromCWD loads state using the current repository git dir.
func LoadFromCWD() (*State, string, error) {
	gitDir, err := GitDir()
	if err != nil {
		return nil, "", err
	}
	s, err := Load(gitDir)
	return s, gitDir, err
}
