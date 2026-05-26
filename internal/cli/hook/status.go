package hook

import (
	"fmt"
	"io"
	"os"

	"github.com/bees-hive/elegant-git/internal/cli/legacy"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/runtime"
	"github.com/spf13/cobra"
)

var statusID = cmdid.ID{Command: "hook", Action: "status"}

func newStatusCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "status",
		Short: "Lists configured hook file paths",
		Long:  "Prints paths of ahead/after hook scripts (new and legacy layouts).",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cliruntime.RunWithWorkflows(cmd, statusID, func() error {
				return statusList(runtime.Workspace{RepoRoot: "."}, cmd.OutOrStdout())
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func statusList(ws runtime.Workspace, w io.Writer) error {
	for _, id := range legacy.AllIDs() {
		legacyName, _ := legacy.IDToLegacy(id)
		for _, hookType := range []string{"ahead", "after"} {
			listIfExists(w, ws.NewHookFile(ws.PersonalHookDir(id), id, hookType))
			listIfExists(w, ws.NewHookFile(ws.CommonHookDir(id), id, hookType))
			if legacyName != "" {
				listIfExists(w, ws.LegacyPersonalHookFile(legacyName, hookType))
				listIfExists(w, ws.LegacyCommonHookFile(legacyName, hookType))
			}
		}
	}
	hookListID := cmdid.ID{Command: "hook", Action: "list"}
	for _, hookType := range []string{"ahead", "after"} {
		listIfExists(w, ws.NewHookFile(ws.PersonalHookDir(statusID), statusID, hookType))
		listIfExists(w, ws.NewHookFile(ws.CommonHookDir(statusID), statusID, hookType))
		listIfExists(w, ws.NewHookFile(ws.PersonalHookDir(hookListID), hookListID, hookType))
		listIfExists(w, ws.NewHookFile(ws.CommonHookDir(hookListID), hookListID, hookType))
	}
	return nil
}

func listIfExists(w io.Writer, path string) {
	if path == "" {
		return
	}
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintln(w, path)
	}
}
