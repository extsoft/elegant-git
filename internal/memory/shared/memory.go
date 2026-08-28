// Package shared implements the user-level elegant-git state file.
package shared

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	SchemaVersion = 2
	stateFileName = "state.json"
)

// State is the on-disk shared memory document.
type State struct {
	SchemaVersion   int                    `json:"schema_version"`
	AcquiredVersion string                 `json:"acquired_version,omitempty"`
	Workspaces      map[string]*Workspace  `json:"workspaces"`
	Repositories    map[string]*Repository `json:"repositories"`
}

// Workspace holds git user identity fields for reuse across repos.
type Workspace struct {
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
	WorkspaceID string   `json:"workspace_id"`
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
// When the on-disk schema is older than SchemaVersion (or still uses legacy
// keys), the file is backed up to path+".bak" and rewritten to the current schema.
func Load() (*State, error) {
	mu.Lock()
	defer mu.Unlock()
	lastLoadHadLegacyKeys = false
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
	var w stateWire
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, err
	}
	if w.SchemaVersion > SchemaVersion {
		return nil, fmt.Errorf("shared memory schema_version %d is newer than supported %d; upgrade elegant-git", w.SchemaVersion, SchemaVersion)
	}
	fromVersion := w.SchemaVersion
	if fromVersion == 0 {
		fromVersion = 1
	}
	s, hadLegacy := decodeState(w)
	lastLoadHadLegacyKeys = hadLegacy
	needsMigrate := fromVersion < SchemaVersion || hadLegacy
	if needsMigrate {
		if hadLegacy {
			recordLegacyMemoryKeys()
		}
		backupPath, err := backupStateFile(p)
		if err != nil {
			return nil, fmt.Errorf("backup shared memory before schema migrate: %w", err)
		}
		if err := writeStateFileLocked(p, s); err != nil {
			return nil, fmt.Errorf("rewrite shared memory after schema migrate (backup at %s): %w", backupPath, err)
		}
		fmt.Fprintf(os.Stderr, "elegant-git: migrated shared memory schema %d → %d (backup: %s)\n",
			fromVersion, SchemaVersion, backupPath)
		lastLoadHadLegacyKeys = false
	}
	cached = s
	cachedAt = p
	return s, nil
}

func emptyState() *State {
	return &State{
		SchemaVersion: SchemaVersion,
		Workspaces:    map[string]*Workspace{},
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
	if err := writeStateFileLocked(p, s); err != nil {
		return err
	}
	lastLoadHadLegacyKeys = false
	cached = s
	cachedAt = p
	return nil
}

func writeStateFileLocked(p string, s *State) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if s.SchemaVersion == 0 || s.SchemaVersion < SchemaVersion {
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
	return nil
}

func backupStateFile(path string) (string, error) {
	backupPath := path + ".bak"
	src, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer src.Close()
	dst, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	if err := dst.Sync(); err != nil {
		return "", err
	}
	return backupPath, nil
}

// Validate reconciles workspace_id ↔ linked_repos in shared state.
func Validate(s *State) error {
	if s == nil {
		return nil
	}
	for repoID, repo := range s.Repositories {
		if repo == nil {
			continue
		}
		ws, ok := s.Workspaces[repo.WorkspaceID]
		if !ok || ws == nil {
			return fmt.Errorf("repository %s references unknown workspace %s", repoID, repo.WorkspaceID)
		}
		if !containsString(ws.LinkedRepos, repoID) {
			ws.LinkedRepos = append(ws.LinkedRepos, repoID)
		}
	}
	for wsID, ws := range s.Workspaces {
		if ws == nil {
			continue
		}
		var fixed []string
		for _, repoID := range ws.LinkedRepos {
			repo, ok := s.Repositories[repoID]
			if !ok || repo == nil {
				continue
			}
			if repo.WorkspaceID == wsID {
				fixed = append(fixed, repoID)
			}
		}
		ws.LinkedRepos = fixed
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

// ErrNotFound indicates a missing workspace or repository.
var ErrNotFound = errors.New("not found")
