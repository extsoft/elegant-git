// Package runtime provides injectable repo layout paths and editor hooks for tests.
package runtime

import (
	"context"
	"path/filepath"

	"github.com/bees-hive/elegant-git/internal/cmdid"
)

type repoLayoutKeyType struct{}

var repoLayoutKey = repoLayoutKeyType{}

// RepoLayout holds repo-root-relative paths for hooks.
type RepoLayout struct {
	RepoRoot string
}

// DefaultRepoLayout uses the current git repository root.
func DefaultRepoLayout() RepoLayout {
	return RepoLayout{RepoRoot: "."}
}

// WithRepoLayout stores layout on ctx.
func WithRepoLayout(ctx context.Context, layout RepoLayout) context.Context {
	return context.WithValue(ctx, repoLayoutKey, layout)
}

// RepoLayoutFromContext returns the repo layout from ctx or a default.
func RepoLayoutFromContext(ctx context.Context) RepoLayout {
	if layout, ok := ctx.Value(repoLayoutKey).(RepoLayout); ok {
		return layout
	}
	return DefaultRepoLayout()
}

const (
	hooksRoot       = ".config/elegant-git/hooks"
	legacyHooksRoot = ".workflows"
)

// CommonHookDir returns <repo>/.config/elegant-git/hooks (flat hook files).
func (layout RepoLayout) CommonHookDir(_ cmdid.ID) string {
	return filepath.Join(layout.RepoRoot, hooksRoot)
}

// PersonalHookDir returns <repo>/.git/.config/elegant-git/hooks (flat hook files).
func (layout RepoLayout) PersonalHookDir(_ cmdid.ID) string {
	return filepath.Join(layout.RepoRoot, ".git", hooksRoot)
}

// HooksDir returns the new-layout hooks directory (common or personal).
func (layout RepoLayout) HooksDir(personal bool) string {
	if personal {
		return layout.PersonalHookDir(cmdid.ID{})
	}
	return layout.CommonHookDir(cmdid.ID{})
}

// LegacyWorkflowsDir returns the legacy .workflows directory (common or personal).
func (layout RepoLayout) LegacyWorkflowsDir(personal bool) string {
	if personal {
		return filepath.Join(layout.RepoRoot, ".git", legacyHooksRoot)
	}
	return filepath.Join(layout.RepoRoot, legacyHooksRoot)
}

// LegacyCommonHookFile returns <repo>/.workflows/<legacy>-<type>.
func (layout RepoLayout) LegacyCommonHookFile(legacyName, hookType string) string {
	return filepath.Join(layout.RepoRoot, legacyHooksRoot, legacyName+"-"+hookType)
}

// LegacyPersonalHookFile returns <repo>/.git/.workflows/<legacy>-<type>.
func (layout RepoLayout) LegacyPersonalHookFile(legacyName, hookType string) string {
	return filepath.Join(layout.RepoRoot, ".git", legacyHooksRoot, legacyName+"-"+hookType)
}

// NewHookFile returns the path for a new-layout hook file.
func (layout RepoLayout) NewHookFile(dir string, id cmdid.ID, hookType string) string {
	return filepath.Join(dir, id.HookFileName(hookType))
}
