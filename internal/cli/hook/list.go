package hook

import (
	"fmt"
	"io"
	"os"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/legacy"
	"github.com/extsoft/elegant-git/internal/runtime"
	"github.com/extsoft/elegant-git/internal/workflows"
	"github.com/spf13/cobra"
)

var listID = cmdid.ID{Command: "hook", Action: "list"}
var statusID = cmdid.ID{Command: "hook", Action: "status"}

func newListCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "Lists configured hook file paths",
		Long:  "Prints paths of ahead/after hook scripts (new and legacy layouts).",
		RunE:  runList,
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func runList(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	workflows.RunAhead(ctx, listID)
	workflows.RunAhead(ctx, statusID)
	defer func() {
		workflows.RunAfter(ctx, statusID)
		workflows.RunAfter(ctx, listID)
	}()
	return listHooks(runtime.RepoLayout{RepoRoot: "."}, cmd.OutOrStdout())
}

func listHooks(ws runtime.RepoLayout, w io.Writer) error {
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
	for _, hookType := range []string{"ahead", "after"} {
		listIfExists(w, ws.NewHookFile(ws.PersonalHookDir(statusID), statusID, hookType))
		listIfExists(w, ws.NewHookFile(ws.CommonHookDir(statusID), statusID, hookType))
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
