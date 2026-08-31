// Package hooks migrates legacy .workflows hook files to the canonical layout.
package hooks

import (
	"io"
	"os"

	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/legacy"
	"github.com/extsoft/elegant-git/internal/runtime"
	"github.com/extsoft/elegant-git/internal/text"
)

// HasLegacy reports whether a legacy .workflows directory exists.
func HasLegacy(ws runtime.RepoLayout, personal bool) bool {
	_, err := os.Stat(ws.LegacyWorkflowsDir(personal))
	return err == nil
}

// Migrate moves legacy hook files to the new layout. Returns repo-relative new and old paths.
func Migrate(ws runtime.RepoLayout, personal, dryRun bool, w io.Writer) (newPaths, oldPaths []string, err error) {
	if w == nil {
		w = io.Discard
	}
	hooksRoot := ws.HooksDir(personal)
	legacyRoot := ws.LegacyWorkflowsDir(personal)

	renamed, moreOld, err := migrateCanonicalLegacyHooks(ws, personal, hooksRoot, dryRun, w)
	if err != nil {
		return nil, nil, err
	}
	newPaths = append(newPaths, renamed...)
	oldPaths = append(oldPaths, moreOld...)

	extraNew, extraOld, err := migrateRemainingLegacyEntries(ws.RepoRoot, legacyRoot, hooksRoot, dryRun, w)
	if err != nil {
		return nil, nil, err
	}
	newPaths = append(newPaths, extraNew...)
	oldPaths = append(oldPaths, extraOld...)

	if _, err := os.Stat(legacyRoot); err == nil {
		oldPaths = append(oldPaths, text.RepoRelPath(ws.RepoRoot, legacyRoot))
	}
	if err := removeLegacyWorkflowsDir(ws.RepoRoot, legacyRoot, dryRun, w); err != nil {
		return nil, nil, err
	}
	return newPaths, oldPaths, nil
}

func migrateCanonicalLegacyHooks(ws runtime.RepoLayout, personal bool, hooksRoot string, dryRun bool, w io.Writer) (newPaths, oldPaths []string, err error) {
	for legacyName, id := range legacy.LegacyToID {
		for _, hookType := range []string{"ahead", "after"} {
			var oldPath string
			if personal {
				oldPath = ws.LegacyPersonalHookFile(legacyName, hookType)
			} else {
				oldPath = ws.LegacyCommonHookFile(legacyName, hookType)
			}
			if _, statErr := os.Stat(oldPath); statErr != nil {
				continue
			}
			newPath := ws.NewHookFile(hooksRoot, id, hookType)
			relOld, relNew, err := movePreservingMode(ws.RepoRoot, oldPath, newPath, dryRun, w)
			if err != nil {
				return nil, nil, err
			}
			oldPaths = append(oldPaths, relOld)
			newPaths = append(newPaths, relNew)
		}
	}
	return newPaths, oldPaths, nil
}

var selfHookActions = []string{"configure", "list", "doctor"}

// HasObjectPrefix reports whether any oldObject-action-{ahead,after} hook files exist.
func HasObjectPrefix(ws runtime.RepoLayout, personal bool, oldObject string, actions []string) bool {
	if len(actions) == 0 {
		actions = selfHookActions
	}
	hooksRoot := ws.HooksDir(personal)
	for _, action := range actions {
		for _, hookType := range []string{"ahead", "after"} {
			p := ws.NewHookFile(hooksRoot, cmdid.ID{Command: oldObject, Action: action}, hookType)
			if _, err := os.Stat(p); err == nil {
				return true
			}
		}
	}
	return false
}

// RenamePrefix moves hook files from oldObject-action-type to newObject-action-type.
func RenamePrefix(ws runtime.RepoLayout, personal bool, oldObject, newObject string, actions []string, w io.Writer) (newPaths, oldPaths []string, err error) {
	if w == nil {
		w = io.Discard
	}
	if len(actions) == 0 {
		actions = selfHookActions
	}
	hooksRoot := ws.HooksDir(personal)
	for _, action := range actions {
		for _, hookType := range []string{"ahead", "after"} {
			oldPath := ws.NewHookFile(hooksRoot, cmdid.ID{Command: oldObject, Action: action}, hookType)
			if _, statErr := os.Stat(oldPath); statErr != nil {
				continue
			}
			newPath := ws.NewHookFile(hooksRoot, cmdid.ID{Command: newObject, Action: action}, hookType)
			relOld, relNew, err := movePreservingMode(ws.RepoRoot, oldPath, newPath, false, w)
			if err != nil {
				return nil, nil, err
			}
			oldPaths = append(oldPaths, relOld)
			newPaths = append(newPaths, relNew)
		}
	}
	return newPaths, oldPaths, nil
}
