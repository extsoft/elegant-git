// Package sources provides shared completion value lists for CLI args and shell completion.
package sources

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/state"
	"github.com/bees-hive/elegant-git/internal/text"
)

func fetchRemotes() {
	if !state.AreThereRemotes() {
		return
	}
	if err := git.Verbose("fetch", "--all"); err != nil {
		text.InfoText("Unable to fetch. The last local revision will be used.")
	}
}

// Refs returns local branches, remote-tracking branches, and tags.
func Refs(_ context.Context) ([]argspec.Choice, error) {
	fetchRemotes()
	var out []argspec.Choice
	out = append(out, forEachRefChoices("refs/heads", "%(refname:short)\t%(upstream:short)")...)
	out = append(out, forEachRefChoices("refs/remotes", "%(refname:short)")...)
	out = append(out, forEachRefChoices("refs/tags", "%(refname:short)")...)
	sort.Slice(out, func(i, j int) bool { return out[i].Value < out[j].Value })
	return out, nil
}

// LocalBranches returns local branch short names.
func LocalBranches(_ context.Context) ([]argspec.Choice, error) {
	choices := forEachRefChoices("refs/heads", "%(refname:short)\t%(upstream:short)")
	sort.Slice(choices, func(i, j int) bool { return choices[i].Value < choices[j].Value })
	return choices, nil
}

// RemoteBranches returns remote-tracking branch refs (origin/foo).
func RemoteBranches(_ context.Context) ([]argspec.Choice, error) {
	fetchRemotes()
	choices := forEachRefChoices("refs/remotes", "%(refname:short)")
	sort.Slice(choices, func(i, j int) bool { return choices[i].Value < choices[j].Value })
	return choices, nil
}

// BranchNamesUnion returns local branch names plus remote-tracking refs.
func BranchNamesUnion(ctx context.Context) ([]argspec.Choice, error) {
	local, err := LocalBranches(ctx)
	if err != nil {
		return nil, err
	}
	remote, err := RemoteBranches(ctx)
	if err != nil {
		return nil, err
	}
	out := append(append([]argspec.Choice{}, local...), remote...)
	sort.Slice(out, func(i, j int) bool { return out[i].Value < out[j].Value })
	return out, nil
}

func forEachRefChoices(pattern, format string) []argspec.Choice {
	raw := git.OutputOK("for-each-ref", "--format="+format, pattern)
	if raw == "" {
		return nil
	}
	var out []argspec.Choice
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		val := parts[0]
		desc := ""
		if len(parts) > 1 {
			desc = parts[1]
		}
		out = append(out, argspec.Choice{Value: val, Description: desc})
	}
	return out
}

// WorkspaceCreateNew is the picker sentinel for creating a workspace during repo configure.
const WorkspaceCreateNew = "[Create new]"

// Workspaces returns workspace display names from shared memory.
func Workspaces(_ context.Context) ([]argspec.Choice, error) {
	s, err := shared.Load()
	if err != nil {
		return nil, err
	}
	var out []argspec.Choice
	for _, p := range shared.ListWorkspaces(s) {
		if p == nil {
			continue
		}
		out = append(out, argspec.Choice{
			Value:       p.Name,
			Description: p.UserName + " <" + p.UserEmail + ">",
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Value < out[j].Value })
	return out, nil
}

// WorkspacesWithCreateNew returns workspaces plus a create-new picker entry.
func WorkspacesWithCreateNew(ctx context.Context) ([]argspec.Choice, error) {
	out, err := Workspaces(ctx)
	if err != nil {
		return nil, err
	}
	out = append(out, argspec.Choice{Value: WorkspaceCreateNew, Description: "Create a new workspace"})
	sort.Slice(out, func(i, j int) bool { return out[i].Value < out[j].Value })
	return out, nil
}

// HookTypes returns ahead/after hook types.
func HookTypes(_ context.Context) ([]argspec.Choice, error) {
	return staticChoices("ahead", "after"), nil
}

// HookLocations returns personal/common hook locations.
func HookLocations(_ context.Context) ([]argspec.Choice, error) {
	return staticChoices("personal", "common"), nil
}

func staticChoices(values ...string) []argspec.Choice {
	out := make([]argspec.Choice, len(values))
	for i, v := range values {
		out[i] = argspec.Choice{Value: v}
	}
	return out
}

// CompletionShells returns supported completion shell names.
func CompletionShells(_ context.Context) ([]argspec.Choice, error) {
	return staticChoices("bash", "zsh", "fish", "powershell"), nil
}

// ReleaseNotesLayouts returns simple/smart release notes layouts.
func ReleaseNotesLayouts(_ context.Context) ([]argspec.Choice, error) {
	return staticChoices("simple", "smart"), nil
}

// HookPaths returns existing hook script paths under repo and personal hook dirs.
func HookPaths(_ context.Context) ([]argspec.Choice, error) {
	dirs := []string{
		filepath.Join(".config", "elegant-git", "hooks"),
		filepath.Join(".git", ".config", "elegant-git", "hooks"),
	}
	seen := map[string]bool{}
	var out []argspec.Choice
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if seen[path] {
				return nil
			}
			seen[path] = true
			out = append(out, argspec.Choice{Value: path})
			return nil
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Value < out[j].Value })
	return out, nil
}

// HookCommandIDs returns canonical object.action command ids (from cli.AllCanonicalCommandIDs).
var hookCommandIDsProvider func() []string

// SetHookCommandIDsProvider registers the canonical command id list (called from cli init).
func SetHookCommandIDsProvider(fn func() []string) {
	hookCommandIDsProvider = fn
}

// HookCommandIDs returns canonical command ids for hook new completion.
func HookCommandIDs(_ context.Context) ([]argspec.Choice, error) {
	if hookCommandIDsProvider == nil {
		return nil, nil
	}
	ids := hookCommandIDsProvider()
	out := make([]argspec.Choice, len(ids))
	for i, id := range ids {
		out[i] = argspec.Choice{Value: id}
	}
	return out, nil
}
