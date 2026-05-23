package hook

import (
	"os"

	"github.com/bees-hive/elegant-git/internal/cli/legacy"
	"github.com/bees-hive/elegant-git/internal/runtime"
	"github.com/bees-hive/elegant-git/internal/text"
)

// MigrateHooks moves legacy hook files to the new layout. Returns repo-relative new and old paths.
func MigrateHooks(ws runtime.Workspace, personal, dryRun bool) (newPaths, oldPaths []string, err error) {
	hooksRoot := ws.HooksDir(personal)
	legacyRoot := ws.LegacyWorkflowsDir(personal)

	renamed, moreOld, err := migrateCanonicalLegacyHooks(ws, personal, hooksRoot, dryRun)
	if err != nil {
		return nil, nil, err
	}
	newPaths = append(newPaths, renamed...)
	oldPaths = append(oldPaths, moreOld...)

	extraNew, extraOld, err := migrateRemainingLegacyEntries(ws.RepoRoot, legacyRoot, hooksRoot, dryRun)
	if err != nil {
		return nil, nil, err
	}
	newPaths = append(newPaths, extraNew...)
	oldPaths = append(oldPaths, extraOld...)

	if _, err := os.Stat(legacyRoot); err == nil {
		oldPaths = append(oldPaths, text.RepoRelPath(ws.RepoRoot, legacyRoot))
	}
	if err := removeLegacyWorkflowsDir(ws.RepoRoot, legacyRoot, dryRun); err != nil {
		return nil, nil, err
	}
	return newPaths, oldPaths, nil
}

func migrateCanonicalLegacyHooks(ws runtime.Workspace, personal bool, hooksRoot string, dryRun bool) (newPaths, oldPaths []string, err error) {
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
			relOld, relNew, err := movePreservingMode(ws.RepoRoot, oldPath, newPath, dryRun)
			if err != nil {
				return nil, nil, err
			}
			oldPaths = append(oldPaths, relOld)
			newPaths = append(newPaths, relNew)
		}
	}
	return newPaths, oldPaths, nil
}
