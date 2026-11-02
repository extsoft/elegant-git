package work

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
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
	for _, rel := range files {
		if skipTracked && tracked[filepath.ToSlash(rel)] {
			kept++
			continue
		}
		dst := filepath.Join(root, rel)
		if err := os.RemoveAll(dst); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.Rename(filepath.Join(aside, rel), dst); err != nil {
			return err
		}
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

func validRelPath(p string) bool {
	if p == "" || path.IsAbs(p) {
		return false
	}
	clean := path.Clean(p)
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}
