package config

import (
	"strings"

	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/state"
)

// SetBranchSourceBranch records the branch that branch was created from.
func SetBranchSourceBranch(branch, source string) error {
	return memrepo.SetBranchSourceFromCWD(branch, source)
}

// ClearBranchSourceBranch removes the recorded source branch for branch.
func ClearBranchSourceBranch(branch string) error {
	return memrepo.ClearBranchSourceFromCWD(branch)
}

// BranchSourceBranch returns the recorded source branch for branch, or the default branch.
func BranchSourceBranch(branch string) string {
	s, gitDir, err := memrepo.LoadFromCWD()
	if err != nil {
		return DefaultBranch()
	}
	recorded := memrepo.BranchSource(s, branch)
	if recorded == "" {
		return DefaultBranch()
	}
	if !state.LocalBranchExists(branch) {
		memrepo.ClearBranchSource(s, branch)
		_ = memrepo.Save(gitDir, s)
		return DefaultBranch()
	}
	if state.RefExists(recorded) {
		return recorded
	}
	fallback := DefaultBranch()
	memrepo.SetBranchSource(s, branch, fallback)
	_ = memrepo.Save(gitDir, s)
	return fallback
}

// FreshestBranchSourceBranch returns the best available ref to rebase onto for branch's source.
func FreshestBranchSourceBranch(branch string) string {
	source := BranchSourceBranch(branch)
	for _, ref := range branchSourceRebaseCandidates(source) {
		if state.RefExists(ref) {
			return ref
		}
	}
	return FreshestDefaultBranch()
}

func branchSourceRebaseCandidates(source string) []string {
	local := localBranchName(source)
	seen := map[string]bool{}
	var out []string
	add := func(ref string) {
		ref = strings.TrimSpace(ref)
		if ref == "" || seen[ref] {
			return
		}
		seen[ref] = true
		out = append(out, ref)
	}

	// 1. Remote tracking branch configured on the local source branch.
	if state.IsThereUpstreamFor(local) {
		add(state.UpstreamOf(local))
	}

	// 2. Remote-tracking ref on the default upstream remote, then the recorded remote ref.
	if state.AreThereRemotes() {
		add(DefaultUpstreamRemote + "/" + local)
		if strings.Contains(source, "/") {
			add(source)
		}
	}

	// 3. Local source branch.
	add(local)
	return out
}

func localBranchName(source string) string {
	if i := strings.Index(source, "/"); i >= 0 {
		return source[i+1:]
	}
	return source
}
