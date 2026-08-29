package shared

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/extsoft/elegant-git/internal/uuidv7"
)

// UpsertRepoInput holds registry fields for a repository.
type UpsertRepoInput struct {
	ID          string
	Name        string
	WorkspaceID string
	CurrentPath string
	OriginURL   string
}

// UpsertRepo creates or updates a repository registry entry.
func UpsertRepo(s *State, in UpsertRepoInput) error {
	if in.WorkspaceID == "" {
		return fmt.Errorf("workspace_id is required")
	}
	if _, err := GetWorkspace(s, in.WorkspaceID); err != nil {
		return err
	}
	id := in.ID
	if id == "" {
		var err error
		id, err = uuidv7.New()
		if err != nil {
			return err
		}
	}
	path := in.CurrentPath
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	existing, ok := s.Repositories[id]
	if ok && existing != nil && existing.WorkspaceID != "" && existing.WorkspaceID != in.WorkspaceID {
		unlinkRepo(s, id, existing.WorkspaceID)
	}
	s.Repositories[id] = &Repository{
		Name:        in.Name,
		WorkspaceID: in.WorkspaceID,
		CurrentPath: path,
		PathHistory: pathHistory(existing),
		OriginURL:   in.OriginURL,
	}
	linkRepo(s, id, in.WorkspaceID)
	return nil
}

func pathHistory(existing *Repository) []string {
	if existing == nil {
		return nil
	}
	return existing.PathHistory
}

func linkRepo(s *State, repoID, workspaceID string) {
	ws := s.Workspaces[workspaceID]
	if ws == nil {
		return
	}
	if !containsString(ws.LinkedRepos, repoID) {
		ws.LinkedRepos = append(ws.LinkedRepos, repoID)
	}
}

func unlinkRepo(s *State, repoID, workspaceID string) {
	ws := s.Workspaces[workspaceID]
	if ws == nil {
		return
	}
	ws.LinkedRepos = removeString(ws.LinkedRepos, repoID)
}

// GetRepo returns a repository by id.
func GetRepo(s *State, id string) (*Repository, error) {
	r, ok := s.Repositories[id]
	if !ok || r == nil {
		return nil, ErrNotFound
	}
	return r, nil
}

// ResolveRepository finds a managed repository by display name or filesystem path.
func ResolveRepository(s *State, arg string) (string, *Repository, error) {
	if arg == "" {
		return "", nil, fmt.Errorf("repository name or path is required")
	}
	absArg := arg
	if abs, err := filepath.Abs(arg); err == nil {
		absArg = abs
	}
	var matched []string
	for id, r := range s.Repositories {
		if r == nil {
			continue
		}
		if r.Name == arg || r.CurrentPath == arg || r.CurrentPath == absArg {
			matched = append(matched, id)
		}
	}
	switch len(matched) {
	case 0:
		return "", nil, fmt.Errorf("repository %q not found", arg)
	case 1:
		return matched[0], s.Repositories[matched[0]], nil
	default:
		return "", nil, fmt.Errorf("repository %q is ambiguous (%d matches); use the full path", arg, len(matched))
	}
}

// ListRepos returns all repositories.
func ListRepos(s *State) map[string]*Repository {
	if s == nil || s.Repositories == nil {
		return map[string]*Repository{}
	}
	return s.Repositories
}

// Relink changes which workspace owns a repository.
func Relink(s *State, repoID, newWorkspaceID string) error {
	repo, err := GetRepo(s, repoID)
	if err != nil {
		return err
	}
	if _, err := GetWorkspace(s, newWorkspaceID); err != nil {
		return err
	}
	old := repo.WorkspaceID
	repo.WorkspaceID = newWorkspaceID
	if old != newWorkspaceID {
		unlinkRepo(s, repoID, old)
		linkRepo(s, repoID, newWorkspaceID)
	}
	return nil
}

// RecordPath updates current_path when the repo is seen at a new location.
func RecordPath(s *State, repoID, absPath string) error {
	repo, err := GetRepo(s, repoID)
	if err != nil {
		return err
	}
	if repo.CurrentPath == absPath {
		return nil
	}
	repo.PathHistory = append(repo.PathHistory, repo.CurrentPath)
	repo.CurrentPath = absPath
	return nil
}

// DeleteRepo removes a repository from the registry.
func DeleteRepo(s *State, repoID string) error {
	repo, err := GetRepo(s, repoID)
	if err != nil {
		return err
	}
	unlinkRepo(s, repoID, repo.WorkspaceID)
	delete(s.Repositories, repoID)
	return nil
}
