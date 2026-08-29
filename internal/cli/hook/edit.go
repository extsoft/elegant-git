package hook

import (
	"fmt"
	"os"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/runtime"
	"github.com/spf13/cobra"
)

var editID = cmdid.ID{Command: "hook", Action: "edit"}

func hookEditSpec(path *string) argspec.Spec {
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInputWithComplete("path", 0, true, "Hook file path", path, nil, sources.HookPaths, true),
	}}
}

func newEditCommand() *cobra.Command {
	var path string
	spec := hookEditSpec(&path)
	c := &cobra.Command{
		Use:   "edit <path>",
		Short: "Opens a hook file in your editor",
		Long:  "Opens the given hook path in core.editor.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, editID, func() error {
				return editRun(cmd, args)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func editRun(cmd *cobra.Command, args []string) error {
	var path string
	if err := argspec.ResolveCmd(cmd, args, hookEditSpec(&path)); err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		cliruntime.ExitWorkflowError(fmt.Sprintf("The '%s' file does not exist.", path))
	}
	return runtime.EditorFromContext(cmd.Context())(path)
}
