package repo

import (
	"fmt"
	"strings"

	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/repoid"
)

// ValidateRepoID warns when per-repo repo_id does not match git config.
func ValidateRepoID(s *State) error {
	if s == nil || s.RepoID == "" {
		return nil
	}
	got, err := repoid.ReadLocal()
	if err != nil || got == "" {
		return nil
	}
	if got != s.RepoID {
		return fmt.Errorf("per-repo memory repo_id %q differs from %s %q", s.RepoID, repoid.Key, got)
	}
	return nil
}

// UnsetLegacyElegantGitKeys removes elegant-git.default-branch and protected-branches from git config.
func UnsetLegacyElegantGitKeys() error {
	for _, key := range []string{"elegant-git.default-branch", "elegant-git.protected-branches"} {
		if err := git.ConfigLocalUnset(key); err != nil {
			return err
		}
	}
	return nil
}

// ReadLegacyElegantGitSettings reads legacy keys from git config before unset.
func ReadLegacyElegantGitSettings() (defaultBranch string, protected []string) {
	def := git.ConfigLocalGet("elegant-git.default-branch")
	if def != "" {
		defaultBranch = def
	}
	prot := git.ConfigLocalGet("elegant-git.protected-branches")
	if prot != "" {
		for _, f := range strings.Fields(prot) {
			protected = append(protected, f)
		}
	}
	return defaultBranch, protected
}
