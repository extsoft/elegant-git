package hook

import (
	"fmt"
	"os"
	"strings"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/cli/legacy"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/deprecation"
	"github.com/bees-hive/elegant-git/internal/runtime"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/bees-hive/elegant-git/internal/workflows"
	"github.com/spf13/cobra"
)

var newID = cmdid.ID{Command: "hook", Action: "new"}

func newNewCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "new <command-id> <ahead|after> <personal|common>",
		Short: "Creates a new hook file",
		Long:  "Creates a hook script under .config/elegant-git/hooks/. command-id is canonical (e.g. work.start) or legacy (e.g. start-work).",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, newID, func() error {
				return newRun(cmd, args)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func newRun(cmd *cobra.Command, args []string) error {
	var commandID, hookType, location string
	if err := argspec.ResolveCmd(cmd, args, argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInput("command-id", 0, true, "Command id (e.g. work.start)", &commandID, nil),
		argspec.PositionalInput("hook-type", 1, true, "Hook type (ahead or after)", &hookType, nil),
		argspec.PositionalInput("location", 2, true, "Hook location (personal or common)", &location, nil),
	}}); err != nil {
		return err
	}
	id, ok := legacy.ParseID(commandID)
	if !ok {
		cliruntime.ExitWorkflowError("Please specify a valid command id (e.g. work.start) or legacy name.")
	}
	if _, ok := legacy.LegacyToID[commandID]; ok {
		deprecation.Record(deprecation.DEP007, "hook new command argument: "+commandID, id.String(), "git elegant hook new "+id.String())
	}
	file, err := workflows.WorkflowsFile(location, id, hookType)
	if err != nil {
		text.ErrorText(err.Error())
		os.Exit(1)
	}
	if _, err := os.Stat(file); err == nil {
		cliruntime.ExitWorkflowError(
			fmt.Sprintf("The '%s' file already exists.", file),
			"Please remove it manually and repeat the command if you need a new one.",
		)
	}
	dir, err := workflows.WorkflowsDirectory(location, id)
	if err != nil {
		text.ErrorText(err.Error())
		os.Exit(1)
	}
	if err := cliruntime.ShellVerbose("mkdir", "-p", dir); err != nil {
		return err
	}
	if err := cliruntime.ShellVerbose("touch", file); err != nil {
		return err
	}
	content := strings.Join([]string{
		"#!/usr/bin/env sh -e",
		fmt.Sprintf("# This script invokes %s of the '%s' execution.", hookType, id.String()),
		"",
	}, "\n")
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		return err
	}
	if err := cliruntime.ShellVerbose("chmod", "+x", file); err != nil {
		return err
	}
	return runtime.EditorFromContext(cmd.Context())(file)
}
