// Package repo implements per-repository elegant-git state inside .git/elegant-git/.
package repo

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/extsoft/elegant-git/internal/git"
)

const (
	SchemaVersion = 2
	dirName       = "elegant-git"
	fileName      = "state.json"
)

// State is the per-repo memory document.
type State struct {
	SchemaVersion     int               `json:"schema_version"`
	RepoID            string            `json:"repo_id"`
	WorkspaceID       string            `json:"workspace_id"`
	DefaultBranch     string            `json:"default_branch"`
	ProtectedBranches []string          `json:"protected_branches"`
	BranchSources     map[string]string `json:"branch_sources,omitempty"`
}

type stateWire struct {
	SchemaVersion     int               `json:"schema_version"`
	RepoID            string            `json:"repo_id"`
	WorkspaceID       string            `json:"workspace_id"`
	ProfileID         string            `json:"profile_id"` // v1
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
// When the on-disk schema is older than SchemaVersion (or still uses legacy
// keys), the file is backed up to path+".bak" and rewritten to the current schema.
func Load(gitDir string) (*State, error) {
	p := Path(gitDir)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{SchemaVersion: SchemaVersion}, nil
		}
		return nil, err
	}
	var w stateWire
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, err
	}
	if w.SchemaVersion > SchemaVersion {
		return nil, fmt.Errorf("per-repo memory schema_version %d is newer than supported %d; upgrade elegant-git", w.SchemaVersion, SchemaVersion)
	}
	fromVersion := w.SchemaVersion
	if fromVersion == 0 {
		fromVersion = 1
	}
	hadLegacy := w.ProfileID != ""
	wsID := w.WorkspaceID
	if wsID == "" {
		wsID = w.ProfileID
	}
	s := &State{
		SchemaVersion:     SchemaVersion,
		RepoID:            w.RepoID,
		WorkspaceID:       wsID,
		DefaultBranch:     w.DefaultBranch,
		ProtectedBranches: w.ProtectedBranches,
		BranchSources:     w.BranchSources,
	}
	needsMigrate := fromVersion < SchemaVersion || hadLegacy
	if needsMigrate {
		if hadLegacy {
			deprecation.Record(
				deprecation.DEP012,
				"shared/per-repo memory keys: profiles, profile_id",
				"workspaces, workspace_id",
				"",
			)
		}
		backupPath, err := backupStateFile(p)
		if err != nil {
			return nil, fmt.Errorf("backup per-repo memory before schema migrate: %w", err)
		}
		if err := writeStateFile(p, s); err != nil {
			return nil, fmt.Errorf("rewrite per-repo memory after schema migrate (backup at %s): %w", backupPath, err)
		}
		fmt.Fprintf(os.Stderr, "elegant-git: migrated per-repo memory schema %d → %d (backup: %s)\n",
			fromVersion, SchemaVersion, backupPath)
	}
	return s, nil
}

// Save atomically writes per-repo state.
func Save(gitDir string, s *State) error {
	return writeStateFile(Path(gitDir), s)
}

func writeStateFile(p string, s *State) error {
	if s.SchemaVersion == 0 || s.SchemaVersion < SchemaVersion {
		s.SchemaVersion = SchemaVersion
	}
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
