// Package state inspects repository state via git.
package state

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/extsoft/elegant-git/internal/git"
)

// IsThereActiveRebase reports whether a rebase is in progress.
func IsThereActiveRebase() bool {
	return rebaseDir("rebase-merge") != "" || rebaseDir("rebase-apply") != ""
}

// RebasingBranch returns the branch name being rebased, or "" if none.
func RebasingBranch() string {
	for _, loc := range []string{"rebase-merge", "rebase-apply"} {
		path := rebaseDir(loc)
		if path == "" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(path, "head-name"))
		if err != nil {
			continue
		}
		ref := strings.TrimSpace(string(data))
		return strings.TrimPrefix(ref, "refs/heads/")
	}
	return ""
}

// LastTag returns the most recent tag by version sort, or "" if none.
func LastTag() string {
	return git.OutputOK("for-each-ref", "--sort", "-version:refname", "--format", "%(refname:short)", "refs/tags", "--count", "1")
}

// IsThereUpstreamFor reports whether branch has an upstream configured.
func IsThereUpstreamFor(branch string) bool {
	_, err := git.Output("rev-parse", "--abbrev-ref", branch+"@{upstream}")
	return err == nil
}

// UpstreamOf returns the upstream branch name for branch (e.g. origin/main), or "" if unset.
func UpstreamOf(branch string) string {
	out, err := git.Output("rev-parse", "--abbrev-ref", branch+"@{upstream}")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// IsRemoteBranch reports whether refs/remotes/<branch> exists.
func IsRemoteBranch(branch string) bool {
	out := git.OutputOK("for-each-ref", "refs/remotes/"+branch)
	return strings.TrimSpace(out) != ""
}

// AreThereRemotes reports whether the repository has any remotes.
func AreThereRemotes() bool {
	out := git.OutputOK("remote")
	return strings.TrimSpace(out) != ""
}

// RefExists reports whether ref resolves in the current repository.
func RefExists(ref string) bool {
	_, err := git.Output("rev-parse", "--verify", "--quiet", ref)
	return err == nil
}

// LocalBranchNames returns short names of local branches.
func LocalBranchNames() ([]string, error) {
	out, err := git.Output("for-each-ref", "--format=%(refname:lstrip=2)", "refs/heads")
	if err != nil {
		return nil, git.ErrorFromOutput(out, err)
	}
	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			names = append(names, line)
		}
	}
	return names, nil
}

// LocalBranchExists reports whether a local branch ref exists.
func LocalBranchExists(name string) bool {
	_, err := git.Output("rev-parse", "--verify", "--quiet", "--abbrev-ref", "--branches=refs/heads", name)
	return err == nil
}

func gitPath(name string) (string, error) {
	out, err := git.Output("rev-parse", "--git-path", name)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func rebaseDir(name string) string {
	path, err := gitPath(name)
	if err != nil || !isRebaseMetadataDir(path) {
		return ""
	}
	return path
}

func isRebaseMetadataDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// ChangesContained reports whether ref's changes are already in target.
// It is false when merge-tree fails, including on Git older than 2.40.
func ChangesContained(target, ref string) bool {
	target = strings.TrimSpace(target)
	ref = strings.TrimSpace(ref)
	if target == "" || ref == "" {
		return false
	}
	mb, err := git.Output("merge-base", target, ref)
	mb = firstLine(mb)
	if err != nil || mb == "" {
		return false
	}
	return changesContainedAt(target, ref, mb)
}

// AlreadyAppliedBase returns the newest first-parent commit on HEAD, after the
// merge base with target, whose changes are already in target. It is empty when
// none match.
func AlreadyAppliedBase(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	mb, err := git.Output("merge-base", target, "HEAD")
	mb = firstLine(mb)
	if err != nil || mb == "" {
		return ""
	}
	out, err := git.Output("rev-list", "--reverse", "--first-parent", mb+"..HEAD")
	if err != nil {
		return ""
	}
	var last string
	for _, commit := range strings.Fields(out) {
		if changesContainedAt(target, commit, mb) {
			last = commit
		}
	}
	return last
}

func changesContainedAt(target, ref, mb string) bool {
	merged, err := git.Output("merge-tree", "--write-tree", "--merge-base="+mb, target, ref)
	if err != nil {
		return false
	}
	merged = firstLine(merged)
	if merged == "" {
		return false
	}
	tree, err := git.Output("rev-parse", target+"^{tree}")
	if err != nil {
		return false
	}
	return merged == firstLine(tree)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}
