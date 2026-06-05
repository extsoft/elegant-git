package shared

import "path/filepath"

// UpdateRepoPath records absPath as the repository location when it changed.
// Returns true when current_path was updated.
func UpdateRepoPath(s *State, repoID, absPath string) (bool, error) {
	repo, err := GetRepo(s, repoID)
	if err != nil {
		return false, err
	}
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return false, err
	}
	if repo.CurrentPath == absPath {
		return false, nil
	}
	if err := RecordPath(s, repoID, absPath); err != nil {
		return false, err
	}
	return true, nil
}
