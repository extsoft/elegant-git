// Package workflows runs optional ahead/after hook scripts for a command.
package workflows

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/bees-hive/elegant-git/internal/cli/legacy"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/deprecation"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/runtime"
	"github.com/bees-hive/elegant-git/internal/text"
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

func workspace(ctx context.Context) runtime.Workspace {
	ws := runtime.FromContext(ctx)
	if ws.RepoRoot == "." || ws.RepoRoot == "" {
		ws.RepoRoot = repoRootFunc()
	}
	return ws
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
	ws := workspace(ctx)
	if runFileIfExists(ws.LegacyPersonalHookFile(legacyName, hookType), true) {
		deprecation.RecordLegacyPersonalHook(ws.LegacyPersonalHookFile(legacyName, hookType))
	}
	if runFileIfExists(ws.LegacyCommonHookFile(legacyName, hookType), true) {
		deprecation.RecordLegacyCommonHook(ws.LegacyCommonHookFile(legacyName, hookType))
	}
}

func runHook(ctx context.Context, id cmdid.ID, hookType string) {
	if Skip {
		return
	}
	ws := workspace(ctx)
	legacyName, _ := legacy.IDToLegacy(id)
	if legacyName == "" {
		legacyName = id.Command + "-" + id.Action
	}
	newPersonal := ws.NewHookFile(ws.PersonalHookDir(id), id, hookType)
	newCommon := ws.NewHookFile(ws.CommonHookDir(id), id, hookType)
	legacyPersonal := ws.LegacyPersonalHookFile(legacyName, hookType)
	legacyCommon := ws.LegacyCommonHookFile(legacyName, hookType)

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
	ws := runtime.Workspace{RepoRoot: repoRootFunc()}
	switch location {
	case "personal":
		return ws.PersonalHookDir(id), nil
	case "common":
		return ws.CommonHookDir(id), nil
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
	ws := runtime.Workspace{RepoRoot: repoRootFunc()}
	return ws.NewHookFile(dir, id, hookType), nil
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
