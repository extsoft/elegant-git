package cli

import (
	"os"
	"os/exec"
	"strings"

	"github.com/bees-hive/elegant-git/internal/exitcode"
	"github.com/bees-hive/elegant-git/internal/git"
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

func requireArg(args []string, index int, message string) {
	if len(args) <= index || args[index] == "" {
		text.ErrorText(message)
		os.Exit(exitcode.EmptyArgument)
	}
}

func exitWorkflowError(messages ...string) {
	for _, m := range messages {
		text.ErrorText(m)
	}
	os.Exit(exitcode.WorkflowError)
}

func shellVerbose(name string, args ...string) error {
	text.CommandText(append([]string{name}, args...)...)
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func openInEditor(path string) error {
	editor := strings.TrimSpace(git.OutputOK("config", "core.editor"))
	if editor == "" {
		editor = "vi"
	}
	text.CommandText(append(strings.Fields(editor), path)...)
	cmd := exec.Command("sh", "-c", editor+" "+shellQuote(path))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func runAcquireRepositoryHooks(fn func() error) error {
	workflows.RunAhead("acquire-repository")
	defer workflows.RunAfter("acquire-repository")
	return fn()
}
