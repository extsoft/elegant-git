// Package workflows runs optional ahead/after hook scripts for a command.
package workflows

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/extsoft/elegant-git/internal/cli/legacy"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/runtime"
	"github.com/extsoft/elegant-git/internal/text"
)

// Skip when true disables ahead/after hook execution (--no-workflows).
var Skip bool

var repoRootFunc = defaultRepoRoot

func defaultRepoRoot() string {
	out := git.OutputOK("rev-parse", "--show-toplevel")
	if out == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "."
		}
		return wd
	}
	return out
}

func repoLayout(ctx context.Context) runtime.RepoLayout {
	layout := runtime.RepoLayoutFromContext(ctx)
	if layout.RepoRoot == "." || layout.RepoRoot == "" {
		layout.RepoRoot = repoRootFunc()
	}
	return layout
}

// RunAhead executes personal then common ahead hooks for id.
func RunAhead(ctx context.Context, id cmdid.ID) {
	runHook(ctx, id, "ahead")
}

// RunAfter executes personal then common after hooks for id.
func RunAfter(ctx context.Context, id cmdid.ID) {
	runHook(ctx, id, "after")
}

// RunAheadCompat runs new id hooks plus legacy-name hooks (DEP-008).
func RunAheadCompat(ctx context.Context, id cmdid.ID, legacyAlso string) {
	RunAhead(ctx, id)
	if legacyAlso != "" {
		runLegacyOnly(ctx, legacyAlso, "ahead")
	}
}

// RunAfterCompat runs new id hooks plus legacy-name hooks.
func RunAfterCompat(ctx context.Context, id cmdid.ID, legacyAlso string) {
	RunAfter(ctx, id)
	if legacyAlso != "" {
		runLegacyOnly(ctx, legacyAlso, "after")
	}
}

func runLegacyOnly(ctx context.Context, legacyName, hookType string) {
	if Skip {
		return
	}
	layout := repoLayout(ctx)
	if runFileIfExists(layout.LegacyPersonalHookFile(legacyName, hookType), true) {
		deprecation.RecordLegacyPersonalHook(layout.LegacyPersonalHookFile(legacyName, hookType))
	}
	if runFileIfExists(layout.LegacyCommonHookFile(legacyName, hookType), true) {
		deprecation.RecordLegacyCommonHook(layout.LegacyCommonHookFile(legacyName, hookType))
	}
}

func runHook(ctx context.Context, id cmdid.ID, hookType string) {
	if Skip {
		return
	}
	layout := repoLayout(ctx)
	legacyName, _ := legacy.IDToLegacy(id)
	if legacyName == "" {
		legacyName = id.Command + "-" + id.Action
	}
	newPersonal := layout.NewHookFile(layout.PersonalHookDir(id), id, hookType)
	newCommon := layout.NewHookFile(layout.CommonHookDir(id), id, hookType)
	legacyPersonal := layout.LegacyPersonalHookFile(legacyName, hookType)
	legacyCommon := layout.LegacyCommonHookFile(legacyName, hookType)

	if fileExists(newPersonal) {
		runFile(newPersonal)
	} else if fileExists(legacyPersonal) {
		runFile(legacyPersonal)
		deprecation.RecordLegacyPersonalHook(legacyPersonal)
	}

	if fileExists(newCommon) {
		runFile(newCommon)
	} else if fileExists(legacyCommon) {
		runFile(legacyCommon)
		deprecation.RecordLegacyCommonHook(legacyCommon)
	}
}

// Prefix returns the repository-relative prefix for workflow paths.
func Prefix(id cmdid.ID) string {
	if id.Command == "repo" && (id.Action == "init" || id.Action == "clone") {
		return ""
	}
	out := git.OutputOK("rev-parse", "--show-cdup")
	return strings.TrimSpace(out)
}

// WorkflowsDirectory returns the directory for personal or common hooks (new layout).
func WorkflowsDirectory(location string, id cmdid.ID) (string, error) {
	layout := runtime.RepoLayout{RepoRoot: repoRootFunc()}
	switch location {
	case "personal":
		return layout.PersonalHookDir(id), nil
	case "common":
		return layout.CommonHookDir(id), nil
	default:
		return "", os.ErrInvalid
	}
}

// WorkflowsFile returns the path to a new-layout workflow hook file.
func WorkflowsFile(location string, id cmdid.ID, hookType string) (string, error) {
	dir, err := WorkflowsDirectory(location, id)
	if err != nil {
		return "", err
	}
	layout := runtime.RepoLayout{RepoRoot: repoRootFunc()}
	return layout.NewHookFile(dir, id, hookType), nil
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func runFileIfExists(path string, _ bool) bool {
	if !fileExists(path) {
		return false
	}
	runFile(path)
	return true
}

func runFile(path string) {
	if path == "" {
		return
	}
	text.CommandText(path)
	cmd := exec.Command("bash", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Dir = repoRootFunc()
	_ = cmd.Run()
}
