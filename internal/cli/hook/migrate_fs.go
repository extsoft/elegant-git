package hook

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bees-hive/elegant-git/internal/text"
)

func movePreservingMode(repoRoot, src, dst string, dryRun bool) (relOld, relNew string, err error) {
	info, err := os.Stat(src)
	if err != nil {
		return "", "", err
	}
	relOld = text.RepoRelPath(repoRoot, src)
	relNew = text.RepoRelPath(repoRoot, dst)
	fmt.Fprintf(os.Stdout, "  %s -> %s\n", relOld, relNew)
	if dryRun {
		return relOld, relNew, nil
	}
	if info.IsDir() {
		if err := os.Rename(src, dst); err == nil {
			return relOld, relNew, nil
		}
		if err := copyTreePreserveMode(src, dst); err != nil {
			return "", "", err
		}
		return relOld, relNew, os.RemoveAll(src)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", "", err
	}
	if err := os.Rename(src, dst); err != nil {
		if err := copyFilePreserveMode(src, dst, info.Mode()); err != nil {
			return "", "", err
		}
		if err := os.Remove(src); err != nil {
			return "", "", err
		}
	} else if err := os.Chmod(dst, info.Mode().Perm()); err != nil {
		return "", "", err
	}
	return relOld, relNew, nil
}

func copyTreePreserveMode(srcRoot, dstRoot string) error {
	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		dst := filepath.Join(dstRoot, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(dst, info.Mode().Perm())
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return copyFilePreserveMode(path, dst, info.Mode())
	})
}

func copyFilePreserveMode(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return os.Chmod(dst, mode.Perm())
}

func migrateRemainingLegacyEntries(repoRoot, legacyRoot, hooksRoot string, dryRun bool) (newPaths, oldPaths []string, err error) {
	if _, err := os.Stat(legacyRoot); err != nil {
		return nil, nil, nil
	}
	entries, err := os.ReadDir(legacyRoot)
	if err != nil {
		return nil, nil, err
	}
	for _, ent := range entries {
		src := filepath.Join(legacyRoot, ent.Name())
		dst := filepath.Join(hooksRoot, ent.Name())
		if _, err := os.Stat(dst); err == nil {
			if ent.IsDir() {
				moreNew, moreOld, err := migrateRemainingLegacyEntries(repoRoot, src, dst, dryRun)
				if err != nil {
					return nil, nil, err
				}
				newPaths = append(newPaths, moreNew...)
				oldPaths = append(oldPaths, moreOld...)
				if !dryRun {
					if err := os.RemoveAll(src); err != nil {
						return nil, nil, err
					}
				}
				continue
			}
			return nil, nil, fmt.Errorf("migrate: %s already exists", dst)
		}
		relOld, relNew, err := movePreservingMode(repoRoot, src, dst, dryRun)
		if err != nil {
			return nil, nil, err
		}
		oldPaths = append(oldPaths, relOld)
		newPaths = append(newPaths, relNew)
	}
	return newPaths, oldPaths, nil
}

func removeLegacyWorkflowsDir(repoRoot, legacyRoot string, dryRun bool) error {
	if _, err := os.Stat(legacyRoot); err != nil {
		return nil
	}
	rel := text.RepoRelPath(repoRoot, legacyRoot)
	fmt.Fprintf(os.Stdout, "  remove %s\n", rel)
	if dryRun {
		return nil
	}
	return os.RemoveAll(legacyRoot)
}
