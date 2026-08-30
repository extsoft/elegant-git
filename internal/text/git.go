package text

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

// SuggestGitAddCommit prints a hint to stage migrated paths explicitly.
func SuggestGitAddCommit(newPaths, oldPaths []string, commitMessage string) {
	SuggestGitAddCommitTo(out, newPaths, oldPaths, commitMessage)
}

// SuggestGitAddCommitTo writes the same hint to w.
func SuggestGitAddCommitTo(w io.Writer, newPaths, oldPaths []string, commitMessage string) {
	if w == nil || (len(newPaths) == 0 && len(oldPaths) == 0) {
		return
	}
	var parts []string
	parts = append(parts, "Consider:")
	if len(newPaths) > 0 {
		add := make([]string, len(newPaths))
		for i, p := range newPaths {
			add[i] = shellQuote(p)
		}
		parts = append(parts, "git add "+strings.Join(add, " "))
	}
	if len(oldPaths) > 0 {
		addu := make([]string, len(oldPaths))
		for i, p := range oldPaths {
			addu[i] = shellQuote(p)
		}
		step := "git add -u " + strings.Join(addu, " ")
		if len(newPaths) > 0 {
			parts = append(parts, "&&", step)
		} else {
			parts = append(parts, step)
		}
	}
	parts = append(parts, "&&", "git commit -m", strconv.Quote(commitMessage))
	fmt.Fprintln(w, strings.Join(parts, " "))
}

func shellQuote(s string) string {
	if strings.ContainsAny(s, " \t'\"$\\*?[]") {
		return strconv.Quote(s)
	}
	return s
}

// RepoRelPath returns a repo-root-relative path for git add hints.
func RepoRelPath(repoRoot, path string) string {
	rel := path
	if repoRoot != "." {
		if r, err := filepath.Rel(repoRoot, path); err == nil {
			rel = r
		}
	}
	return strings.TrimPrefix(filepath.ToSlash(rel), "./")
}
