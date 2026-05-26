package shared

import (
	"os"
	"path/filepath"

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
	if _, err := GetRepo(s, repoID); err != nil {
		return nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cwd, err = filepath.Abs(cwd)
	if err != nil {
		return err
	}
	if err := RecordPath(s, repoID, cwd); err != nil {
		return err
	}
	return Save(s)
}
