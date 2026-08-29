package repo

import (
	"strings"

	"github.com/extsoft/elegant-git/internal/git"
)

const defaultUpstreamRemote = "origin"

const (
	defaultBranchDefault     = "main"
	protectedBranchesDefault = "main"
)

// DefaultBranchName returns default branch from per-repo memory or default.
func DefaultBranchName(s *State) string {
	if s != nil && s.DefaultBranch != "" {
		return s.DefaultBranch
	}
	return defaultBranchDefault
}

// ProtectedBranchesList returns protected branch names from per-repo memory.
func ProtectedBranchesList(s *State) []string {
	if s != nil && len(s.ProtectedBranches) > 0 {
		return s.ProtectedBranches
	}
	return []string{protectedBranchesDefault}
}

// ProtectedBranchesString returns space-separated protected branches.
func ProtectedBranchesString(s *State) string {
	return strings.Join(ProtectedBranchesList(s), " ")
}

// IsBranchProtected reports whether name is protected per per-repo memory.
func IsBranchProtected(s *State, name string) bool {
	for _, b := range ProtectedBranchesList(s) {
		if b == name {
			return true
		}
	}
	return false
}

// FreshestDefaultBranch returns remote tracking branch when remotes exist.
func FreshestDefaultBranch(s *State) string {
	def := DefaultBranchName(s)
	if strings.TrimSpace(git.OutputOK("remote")) == "" {
		return def
	}
	return defaultUpstreamRemote + "/" + def
}
