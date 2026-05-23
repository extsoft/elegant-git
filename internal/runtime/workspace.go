// Package runtime provides injectable workspace paths and editor hooks for tests.
package runtime

import (
	"context"
	"path/filepath"

	"github.com/bees-hive/elegant-git/internal/cmdid"
)

type workspaceKeyType struct{}

var workspaceKey = workspaceKeyType{}

// Workspace holds repo-root-relative paths for hooks.
type Workspace struct {
	RepoRoot string
}

// DefaultWorkspace uses the current git repository root.
func DefaultWorkspace() Workspace {
	return Workspace{RepoRoot: "."}
}

// WithWorkspace stores ws on ctx.
func WithWorkspace(ctx context.Context, ws Workspace) context.Context {
	return context.WithValue(ctx, workspaceKey, ws)
}

// FromContext returns the workspace from ctx or a default.
func FromContext(ctx context.Context) Workspace {
	if ws, ok := ctx.Value(workspaceKey).(Workspace); ok {
		return ws
	}
	return DefaultWorkspace()
}

const (
	hooksRoot       = ".config/elegant-git/hooks"
	legacyHooksRoot = ".workflows"
)

// CommonHookDir returns <repo>/.config/elegant-git/hooks (flat hook files).
func (ws Workspace) CommonHookDir(_ cmdid.ID) string {
	return filepath.Join(ws.RepoRoot, hooksRoot)
}

// PersonalHookDir returns <repo>/.git/.config/elegant-git/hooks (flat hook files).
func (ws Workspace) PersonalHookDir(_ cmdid.ID) string {
	return filepath.Join(ws.RepoRoot, ".git", hooksRoot)
}

// HooksDir returns the new-layout hooks directory (common or personal).
func (ws Workspace) HooksDir(personal bool) string {
	if personal {
		return ws.PersonalHookDir(cmdid.ID{})
	}
	return ws.CommonHookDir(cmdid.ID{})
}

// LegacyWorkflowsDir returns the legacy .workflows directory (common or personal).
func (ws Workspace) LegacyWorkflowsDir(personal bool) string {
	if personal {
		return filepath.Join(ws.RepoRoot, ".git", legacyHooksRoot)
	}
	return filepath.Join(ws.RepoRoot, legacyHooksRoot)
}

// LegacyCommonHookFile returns <repo>/.workflows/<legacy>-<type>.
func (ws Workspace) LegacyCommonHookFile(legacyName, hookType string) string {
	return filepath.Join(ws.RepoRoot, legacyHooksRoot, legacyName+"-"+hookType)
}

// LegacyPersonalHookFile returns <repo>/.git/.workflows/<legacy>-<type>.
func (ws Workspace) LegacyPersonalHookFile(legacyName, hookType string) string {
	return filepath.Join(ws.RepoRoot, ".git", legacyHooksRoot, legacyName+"-"+hookType)
}

// NewHookFile returns the path for a new-layout hook file.
func (ws Workspace) NewHookFile(dir string, id cmdid.ID, hookType string) string {
	return filepath.Join(dir, id.HookFileName(hookType))
}
