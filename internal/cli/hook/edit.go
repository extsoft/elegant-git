package hook

import (
	"fmt"
	"os"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/runtime"
	"github.com/spf13/cobra"
)

var editID = cmdid.ID{Command: "hook", Action: "edit"}

func newEditCommand() *cobra.Command {
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
	return c
}

func editRun(cmd *cobra.Command, args []string) error {
	var path string
	if err := argspec.ResolveCmd(cmd, args, argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInput("path", 0, true, "Hook file path", &path, nil),
	}}); err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		cliruntime.ExitWorkflowError(fmt.Sprintf("The '%s' file does not exist.", path))
	}
	return runtime.EditorFromContext(cmd.Context())(path)
}
