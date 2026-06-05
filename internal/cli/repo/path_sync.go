package repo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bees-hive/elegant-git/internal/memory/repoid"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
)

// syncRegistryPath updates shared memory when a registered repo is at path.
// An empty path uses the current working directory.
// When requireRegistered is false, unknown repo-id or registry entries are ignored.
func syncRegistryPath(path string, requireRegistered bool) (changed bool, absPath string, err error) {
	repoID, err := repoid.ReadLocal()
	if err != nil {
		return false, "", err
	}
	if repoID == "" {
		if requireRegistered {
			return false, "", fmt.Errorf("not a configured elegant-git repository")
		}
		return false, "", nil
	}
	if path == "" {
		path, err = os.Getwd()
		if err != nil {
			return false, "", err
		}
	}
	absPath, err = filepath.Abs(path)
	if err != nil {
		return false, "", err
	}
	if _, err := os.Stat(absPath); err != nil {
		return false, "", fmt.Errorf("path does not exist: %s", absPath)
	}
	s, err := shared.Load()
	if err != nil {
		return false, "", err
	}
	changed, err = shared.UpdateRepoPath(s, repoID, absPath)
	if err != nil {
		if requireRegistered {
			return false, "", fmt.Errorf("not a configured elegant-git repository")
		}
		return false, "", nil
	}
	if !changed {
		return false, absPath, nil
	}
	return true, absPath, shared.Save(s)
}
