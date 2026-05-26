package shared

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bees-hive/elegant-git/internal/uuidv7"
)

// UpsertRepoInput holds registry fields for a repository.
type UpsertRepoInput struct {
	ID          string
	Name        string
	ProfileID   string
	CurrentPath string
	OriginURL   string
}

// UpsertRepo creates or updates a repository registry entry.
func UpsertRepo(s *State, in UpsertRepoInput) error {
	if in.ProfileID == "" {
		return fmt.Errorf("profile_id is required")
	}
	if _, err := GetProfile(s, in.ProfileID); err != nil {
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
	if ok && existing != nil && existing.ProfileID != "" && existing.ProfileID != in.ProfileID {
		unlinkRepo(s, id, existing.ProfileID)
	}
	s.Repositories[id] = &Repository{
		Name:        in.Name,
		ProfileID:   in.ProfileID,
		CurrentPath: path,
		PathHistory: pathHistory(existing),
		OriginURL:   in.OriginURL,
	}
	linkRepo(s, id, in.ProfileID)
	return nil
}

func pathHistory(existing *Repository) []string {
	if existing == nil {
		return nil
	}
	return existing.PathHistory
}

func linkRepo(s *State, repoID, profileID string) {
	p := s.Profiles[profileID]
	if p == nil {
		return
	}
	if !containsString(p.LinkedRepos, repoID) {
		p.LinkedRepos = append(p.LinkedRepos, repoID)
	}
}

func unlinkRepo(s *State, repoID, profileID string) {
	p := s.Profiles[profileID]
	if p == nil {
		return
	}
	p.LinkedRepos = removeString(p.LinkedRepos, repoID)
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

// Relink changes which profile owns a repository.
func Relink(s *State, repoID, newProfileID string) error {
	repo, err := GetRepo(s, repoID)
	if err != nil {
		return err
	}
	if _, err := GetProfile(s, newProfileID); err != nil {
		return err
	}
	old := repo.ProfileID
	repo.ProfileID = newProfileID
	if old != newProfileID {
		unlinkRepo(s, repoID, old)
		linkRepo(s, repoID, newProfileID)
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
	unlinkRepo(s, repoID, repo.ProfileID)
	delete(s.Repositories, repoID)
	return nil
}
