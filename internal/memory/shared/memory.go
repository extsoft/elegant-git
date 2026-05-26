// Package shared implements the user-level elegant-git state file.
package shared

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	SchemaVersion = 1
	stateFileName = "state.json"
)

// State is the on-disk shared memory document.
type State struct {
	SchemaVersion int                    `json:"schema_version"`
	Profiles      map[string]*Profile    `json:"profiles"`
	Repositories  map[string]*Repository `json:"repositories"`
}

// Profile holds git user identity fields for reuse across repos.
type Profile struct {
	Name        string   `json:"name"`
	UserName    string   `json:"user_name"`
	UserEmail   string   `json:"user_email"`
	SigningKey  string   `json:"signing_key,omitempty"`
	Editor      string   `json:"editor,omitempty"`
	GPGProgram  string   `json:"gpg_program,omitempty"`
	LinkedRepos []string `json:"linked_repos"`
}

// Repository is a managed repo registry entry.
type Repository struct {
	Name        string   `json:"name"`
	ProfileID   string   `json:"profile_id"`
	CurrentPath string   `json:"current_path"`
	PathHistory []string `json:"path_history,omitempty"`
	OriginURL   string   `json:"origin_url,omitempty"`
}

var (
	mu       sync.Mutex
	cached   *State
	cachedAt string
)

// Path returns the shared state file path.
func Path() (string, error) {
	if override := os.Getenv("ELEGANT_GIT_STATE_FILE"); override != "" {
		return override, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "elegant-git", stateFileName), nil
}

// Load reads and parses the shared state file.
func Load() (*State, error) {
	mu.Lock()
	defer mu.Unlock()
	p, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return emptyState(), nil
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
	if s.Profiles == nil {
		s.Profiles = map[string]*Profile{}
	}
	if s.Repositories == nil {
		s.Repositories = map[string]*Repository{}
	}
	cached = &s
	cachedAt = p
	return &s, nil
}

func emptyState() *State {
	return &State{
		SchemaVersion: SchemaVersion,
		Profiles:      map[string]*Profile{},
		Repositories:  map[string]*Repository{},
	}
}

// Save atomically writes state to disk.
func Save(s *State) error {
	mu.Lock()
	defer mu.Unlock()
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if s.SchemaVersion == 0 {
		s.SchemaVersion = SchemaVersion
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
	cached = s
	cachedAt = p
	return nil
}

// Validate reconciles profile_id ↔ linked_repos in shared state.
func Validate(s *State) error {
	if s == nil {
		return nil
	}
	for repoID, repo := range s.Repositories {
		if repo == nil {
			continue
		}
		p, ok := s.Profiles[repo.ProfileID]
		if !ok || p == nil {
			return fmt.Errorf("repository %s references unknown profile %s", repoID, repo.ProfileID)
		}
		if !containsString(p.LinkedRepos, repoID) {
			p.LinkedRepos = append(p.LinkedRepos, repoID)
		}
	}
	for profID, p := range s.Profiles {
		if p == nil {
			continue
		}
		var fixed []string
		for _, repoID := range p.LinkedRepos {
			repo, ok := s.Repositories[repoID]
			if !ok || repo == nil {
				continue
			}
			if repo.ProfileID == profID {
				fixed = append(fixed, repoID)
			}
		}
		p.LinkedRepos = fixed
	}
	return nil
}

func containsString(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func removeString(ss []string, s string) []string {
	var out []string
	for _, x := range ss {
		if x != s {
			out = append(out, x)
		}
	}
	return out
}

// ErrNotFound indicates a missing profile or repository.
var ErrNotFound = errors.New("not found")
