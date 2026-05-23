package hook

import (
	"fmt"
	"os"

	"github.com/bees-hive/elegant-git/internal/cli/legacy"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/runtime"
	"github.com/spf13/cobra"
)

var listID = cmdid.ID{Command: "hook", Action: "list"}

func newListCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "Lists configured hook file paths",
		Long:  "Prints paths of ahead/after hook scripts (new and legacy layouts).",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cliruntime.RunWithWorkflows(cmd, listID, listRun)
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func listRun() error {
	ws := runtime.Workspace{RepoRoot: "."}
	for _, id := range legacy.AllIDs() {
		legacyName, _ := legacy.IDToLegacy(id)
		for _, hookType := range []string{"ahead", "after"} {
			listIfExists(ws.NewHookFile(ws.PersonalHookDir(id), id, hookType))
			listIfExists(ws.NewHookFile(ws.CommonHookDir(id), id, hookType))
			if legacyName != "" {
				listIfExists(ws.LegacyPersonalHookFile(legacyName, hookType))
				listIfExists(ws.LegacyCommonHookFile(legacyName, hookType))
			}
		}
	}
	return nil
}

func listIfExists(path string) {
	if path == "" {
		return
	}
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintln(os.Stdout, path)
	}
}
