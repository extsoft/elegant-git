package hook

import (
	"io"
	"os"
	"path/filepath"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/statefmt"
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
		Long:  "Prints ahead/after hook scripts (new and legacy layouts) as nested blocks.",
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
	var items []statefmt.Item
	for _, id := range legacy.AllIDs() {
		legacyName, _ := legacy.IDToLegacy(id)
		for _, hookType := range []string{"ahead", "after"} {
			appendHook(&items, ws.NewHookFile(ws.PersonalHookDir(id), id, hookType), "personal")
			appendHook(&items, ws.NewHookFile(ws.CommonHookDir(id), id, hookType), "common")
			if legacyName != "" {
				appendHook(&items, ws.LegacyPersonalHookFile(legacyName, hookType), "legacy personal")
				appendHook(&items, ws.LegacyCommonHookFile(legacyName, hookType), "legacy common")
			}
		}
	}
	for _, hookType := range []string{"ahead", "after"} {
		appendHook(&items, ws.NewHookFile(ws.PersonalHookDir(statusID), statusID, hookType), "personal")
		appendHook(&items, ws.NewHookFile(ws.CommonHookDir(statusID), statusID, hookType), "common")
	}
	statefmt.PrintCatalog(w, "", items)
	return nil
}

func appendHook(items *[]statefmt.Item, path, scope string) {
	if path == "" {
		return
	}
	if _, err := os.Stat(path); err != nil {
		return
	}
	*items = append(*items, statefmt.Item{
		Heading: filepath.Base(path),
		Fields: []statefmt.Field{
			{Key: "scope", Value: scope},
			{Key: "path", Value: path},
			{Key: "explore", Value: "eg hook edit " + path},
		},
	})
}
