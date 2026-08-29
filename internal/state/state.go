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
