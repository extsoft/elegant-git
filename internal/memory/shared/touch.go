package shared

import (
	"os"

	"github.com/bees-hive/elegant-git/internal/memory/repoid"
)

// TouchCurrentRepo updates registry path when repo-id is known and cwd changed.
func TouchCurrentRepo() error {
	repoID, err := repoid.ReadLocal()
	if err != nil || repoID == "" {
		return nil
	}
	s, err := Load()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	changed, err := UpdateRepoPath(s, repoID, cwd)
	if err != nil {
		return nil
	}
	if !changed {
		return nil
	}
	return Save(s)
}
