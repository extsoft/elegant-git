package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/bees-hive/elegant-git/internal/workflows"
	"github.com/spf13/cobra"
)

func newMakeWorkflowCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithWorkflows(cmd, func() error {
				return makeWorkflowRun(args)
			})
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func makeWorkflowRun(args []string) error {
	requireArg(args, 0, "Please specify a name of Elegant Git command")
	requireArg(args, 1, "Please specify a type of the workflow")
	requireArg(args, 2, "Please specify a location of the workflow")

	command := args[0]
	hookType := args[1]
	location := args[2]

	file, err := workflows.WorkflowsFile(location, command, hookType)
	if err != nil {
		text.ErrorText(err.Error())
		os.Exit(1)
	}

	if _, err := os.Stat(file); err == nil {
		exitWorkflowError(
			fmt.Sprintf("The '%s' file already exists.", file),
			"Please remove it manually and repeat the command if you need a new one.",
		)
	}

	dir, err := workflows.WorkflowsDirectory(location, command)
	if err != nil {
		text.ErrorText(err.Error())
		os.Exit(1)
	}

	if err := shellVerbose("mkdir", "-p", dir); err != nil {
		return err
	}
	if err := shellVerbose("touch", file); err != nil {
		return err
	}

	content := strings.Join([]string{
		"#!/usr/bin/env sh -e",
		fmt.Sprintf("# This script invokes %s of the '%s' execution.", hookType, command),
		"",
	}, "\n")
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		return err
	}
	if err := shellVerbose("chmod", "+x", file); err != nil {
		return err
	}
	return openInEditor(file)
}
