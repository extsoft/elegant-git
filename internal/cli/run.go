package cli

import (
	"os"

	"github.com/bees-hive/elegant-git/internal/exitcode"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/bees-hive/elegant-git/internal/workflows"
	"github.com/spf13/cobra"
)

func runWithWorkflows(cmd *cobra.Command, fn func() error) error {
	workflows.RunAhead(cmd.Name())
	defer workflows.RunAfter(cmd.Name())
	return fn()
}

func requireArgs(args []string, message string) {
	if len(args) == 0 || args[0] == "" {
		text.ErrorText(message)
		os.Exit(exitcode.EmptyArgument)
	}
}

func runAcquireRepositoryHooks(fn func() error) error {
	workflows.RunAhead("acquire-repository")
	defer workflows.RunAfter("acquire-repository")
	return fn()
}
