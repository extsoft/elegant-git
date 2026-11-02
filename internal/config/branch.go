package config

import (
	"strings"

	"github.com/extsoft/elegant-git/internal/git"
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
	candidate := ""
	for _, ref := range branchSourceRebaseCandidates(source) {
		if state.RefExists(ref) {
			candidate = ref
			break
		}
	}
	if candidate == "" {
		return FreshestDefaultBranch()
	}
	candidate = preferLocalSource(source, candidate)
	if sourceDelivered(source, candidate) {
		return FreshestDefaultBranch()
	}
	return candidate
}

func preferLocalSource(source, candidate string) string {
	if localBranchName(candidate) == candidate {
		return candidate
	}
	local := localBranchName(source)
	if local == "" || !state.LocalBranchExists(local) || !state.RefExists(local) {
		return candidate
	}
	if sameCommit(local, candidate) || !state.ChangesContained(local, candidate) {
		return candidate
	}
	return local
}

func sourceDelivered(source, candidate string) bool {
	local := localBranchName(source)
	if local == "" || local == DefaultBranch() {
		return false
	}
	if upstreamGone(local) {
		return true
	}
	def := FreshestDefaultBranch()
	if candidate == def || sameCommit(candidate, def) {
		return false
	}
	return state.ChangesContained(def, candidate)
}

func upstreamGone(local string) bool {
	merge, err := git.Output("config", "--get", "branch."+local+".merge")
	if err != nil || strings.TrimSpace(merge) == "" {
		return false
	}
	return !state.IsThereUpstreamFor(local)
}

func sameCommit(a, b string) bool {
	left, err1 := git.Output("rev-parse", a)
	right, err2 := git.Output("rev-parse", b)
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if err1 != nil || err2 != nil || left == "" || right == "" {
		return false
	}
	return left == right
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
	remote, rest := splitRemote(source)
	if remote == "" {
		return source
	}
	return rest
}

func splitRemote(ref string) (string, string) {
	i := strings.Index(ref, "/")
	if i <= 0 || i == len(ref)-1 {
		return "", ref
	}
	remote := ref[:i]
	if !remoteKnown(remote) {
		return "", ref
	}
	return remote, ref[i+1:]
}

func remoteKnown(name string) bool {
	for _, remote := range strings.Split(git.OutputOK("remote"), "\n") {
		if strings.TrimSpace(remote) == name {
			return true
		}
	}
	return false
}
