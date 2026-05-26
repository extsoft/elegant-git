// Package repoid manages elegant-git.repo-id in local git config.
package repoid

import (
	"fmt"
	"strings"

	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/uuidv7"
)

// Key is the git config key for repository UUID.
const Key = "elegant-git.repo-id"

// ReadLocal returns the repo id from local git config, or empty if unset.
func ReadLocal() (string, error) {
	out, err := git.Output("config", "--local", "--get", Key)
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

// EnsureLocal stamps a new UUIDv7 when missing and returns the repo id.
func EnsureLocal() (string, error) {
	id, err := ReadLocal()
	if err != nil {
		return "", err
	}
	if id != "" {
		return id, nil
	}
	id, err = uuidv7.New()
	if err != nil {
		return "", err
	}
	if err := git.ConfigLocalSet(Key, id); err != nil {
		return "", err
	}
	return id, nil
}

// StampLocal writes id to local git config.
func StampLocal(id string) error {
	if id == "" {
		return fmt.Errorf("empty repo id")
	}
	return git.ConfigLocalSet(Key, id)
}
