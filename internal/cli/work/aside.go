package work

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
)

const asideHint = "Files moved aside come back when the rebase finishes through eg work sync, push, accept, or polish."

func upstreamFor(base, target string) string {
	if strings.TrimSpace(base) != "" {
		return base
	}
	return target
}

func collidingPaths(upstream, target string) ([]string, error) {
	replay, err := git.Output("log", "-z", "--format=", "--name-only", "--diff-filter=A", "--no-renames", "--no-merges", upstream+"..HEAD")
	if err != nil {
		return nil, err
	}
	checkout, err := git.Output("diff", "-z", "--name-only", "--diff-filter=A", "--no-renames", "HEAD", target)
	if err != nil {
		return nil, err
	}
	added := map[string]struct{}{}
	for _, p := range splitGitPaths(replay) {
		added[p] = struct{}{}
	}
	for _, p := range splitGitPaths(checkout) {
		added[p] = struct{}{}
	}
	if len(added) == 0 {
		return nil, nil
	}
	tracked, err := trackedPaths()
	if err != nil {
		return nil, err
	}
	root, err := workRoot()
	if err != nil {
		return nil, err
	}
	var out []string
	for p := range added {
		if !validRelPath(p) || tracked[p] {
			continue
		}
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(p))); err != nil {
			continue
		}
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

func moveAside(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	root, err := workRoot()
	if err != nil {
		return err
	}
	gitDir, err := memrepo.GitDir()
	if err != nil {
		return err
	}
	aside := filepath.Join(gitDir, "elegant-git", "aside")
	var moved []string
	for _, p := range paths {
		dst := filepath.Join(aside, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			err = rollbackAside(root, aside, moved, err)
			return err
		}
		if _, err := os.Lstat(dst); err == nil {
			return rollbackAside(root, aside, moved, fmt.Errorf("refusing to move %s aside because %s already exists. Recovery is required", filepath.ToSlash(p), aside))
		} else if err != nil && !os.IsNotExist(err) {
			return rollbackAside(root, aside, moved, err)
		}
		if err := os.Rename(filepath.Join(root, filepath.FromSlash(p)), dst); err != nil {
			return rollbackAside(root, aside, moved, err)
		}
		moved = append(moved, p)
	}
	text.InfoText(fmt.Sprintf("Moved %d files aside to .git/elegant-git/aside until the rebase finishes.", len(moved)))
	return nil
}

func restoreAside() error {
	return restoreAsideFiles(false)
}

func restoreUntrackedAside() error {
	return restoreAsideFiles(true)
}

func restoreAsideFiles(skipTracked bool) error {
	gitDir, err := memrepo.GitDir()
	if err != nil {
		return nil
	}
	aside := filepath.Join(gitDir, "elegant-git", "aside")
	info, err := os.Lstat(aside)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", aside)
	}
	root, err := workRoot()
	if err != nil {
		return err
	}
	var tracked map[string]bool
	if skipTracked {
		tracked, err = trackedPaths()
		if err != nil {
			return err
		}
	}
	var files []string
	err = filepath.WalkDir(aside, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(aside, p)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return err
	}
	restored := 0
	kept := 0
	var pending []string
	for _, rel := range files {
		if skipTracked && tracked[filepath.ToSlash(rel)] {
			kept++
			continue
		}
		if err := checkAsideDestination(root, aside, rel); err != nil {
			return err
		}
		pending = append(pending, rel)
	}
	var moved []string
	for _, rel := range pending {
		dst := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return undoAsideRestore(root, aside, moved, err)
		}
		if err := os.Rename(filepath.Join(aside, rel), dst); err != nil {
			return undoAsideRestore(root, aside, moved, err)
		}
		moved = append(moved, rel)
		restored++
	}
	if kept == 0 {
		if err := os.RemoveAll(aside); err != nil {
			return err
		}
	}
	if restored > 0 {
		text.InfoText(fmt.Sprintf("Restored %d files from .git/elegant-git/aside.", restored))
	}
	return nil
}

func checkAsideDestination(root, aside, rel string) error {
	cleaned := filepath.Clean(rel)
	slash := filepath.ToSlash(rel)
	if cleaned == "." || filepath.IsAbs(cleaned) || escapesWorktree(cleaned) {
		return fmt.Errorf("refusing to restore %s because it leaves the worktree; the aside copy was kept at %s. Recovery is required", slash, aside)
	}
	cur := root
	parts := strings.Split(cleaned, string(os.PathSeparator))
	for i, part := range parts {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("refusing to restore %s because it leaves the worktree; the aside copy was kept at %s. Recovery is required", slash, aside)
		}
		cur = filepath.Join(cur, part)
		info, err := os.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link := filepath.ToSlash(filepath.Join(parts[:i+1]...))
			return fmt.Errorf("refusing to restore %s because %s is a symlink; the aside copy was kept at %s. Recovery is required", slash, link, aside)
		}
		if i == len(parts)-1 {
			return fmt.Errorf("refusing to restore %s because it already exists; the aside copy was kept at %s", slash, aside)
		}
		if !info.IsDir() {
			parent := filepath.ToSlash(filepath.Join(parts[:i+1]...))
			return fmt.Errorf("refusing to restore %s because %s is not a directory; the aside copy was kept at %s. Recovery is required", slash, parent, aside)
		}
	}
	return nil
}

func escapesWorktree(cleaned string) bool {
	return cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator))
}

func undoAsideRestore(root, aside string, moved []string, cause error) error {
	var failed []string
	for i := len(moved) - 1; i >= 0; i-- {
		rel := moved[i]
		src := filepath.Join(root, rel)
		dst := filepath.Join(aside, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			failed = append(failed, rel)
			continue
		}
		if err := os.Rename(src, dst); err != nil {
			failed = append(failed, rel)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("%w; could not return %s to the aside. Recovery is required", cause, strings.Join(failed, ", "))
	}
	return cause
}

func rollbackAside(root, aside string, moved []string, cause error) error {
	var failed []string
	for _, p := range moved {
		dst := filepath.Join(root, filepath.FromSlash(p))
		src := filepath.Join(aside, filepath.FromSlash(p))
		if err := os.Rename(src, dst); err != nil {
			failed = append(failed, p)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("move files aside: %w; could not restore %s", cause, strings.Join(failed, ", "))
	}
	return cause
}

func trackedPaths() (map[string]bool, error) {
	out, err := git.Output("ls-files", "-z")
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			set[p] = true
		}
	}
	return set, nil
}

func workRoot() (string, error) {
	out, err := git.Output("rev-parse", "--show-toplevel")
	out = strings.TrimSpace(out)
	if err != nil || out == "" {
		return "", fmt.Errorf("not a git repository")
	}
	return out, nil
}

func splitGitPaths(out string) []string {
	var paths []string
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}

func ignoredOccupant(paths []string) (string, error) {
	root, err := workRoot()
	if err != nil {
		return "", err
	}
	for _, p := range paths {
		if !validRelPath(p) {
			continue
		}
		full := filepath.Join(root, filepath.FromSlash(p))
		if _, err := os.Lstat(full); err != nil {
			continue
		}
		if _, err := git.Output("-C", root, "check-ignore", "-q", "--", p); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
				continue
			}
			return "", err
		}
		return p, nil
	}
	return "", nil
}

func activeRebaseAdds() ([]string, error) {
	if !state.IsThereActiveRebase() {
		return nil, nil
	}
	todo, err := git.Output("rev-parse", "--git-path", "rebase-merge/git-rebase-todo")
	todo = strings.TrimSpace(todo)
	if err != nil || todo == "" {
		return nil, nil
	}
	data, err := os.ReadFile(todo)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var adds []string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || (fields[0] != "pick" && fields[0] != "p") {
			continue
		}
		out, err := git.Output("diff-tree", "--no-commit-id", "--name-only", "--diff-filter=A", "-r", "-z", fields[1])
		if err != nil {
			return nil, err
		}
		adds = append(adds, splitGitPaths(out)...)
	}
	return adds, nil
}

func validRelPath(p string) bool {
	if p == "" || path.IsAbs(p) {
		return false
	}
	clean := path.Clean(p)
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}
