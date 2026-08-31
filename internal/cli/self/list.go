package self

import (
	"fmt"
	"io"
	"os"

	"github.com/extsoft/elegant-git/internal/cli/statefmt"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/version"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show Elegant Git and global Git installation state",
		Long:  "Prints shared memory paths, workspace and repository counts, and the global Git identity.",
		RunE:  runList,
	}
}

func runList(cmd *cobra.Command, _ []string) error {
	return printSelfList(cmd.OutOrStdout())
}

func printSelfList(w io.Writer) error {
	statefmt.PrintHeading(w, "", "Version")
	fmt.Fprintf(w, "  %s\n", version.Version)
	fmt.Fprintln(w)

	sharedPath, err := shared.Path()
	if err != nil {
		return err
	}
	statefmt.PrintHeading(w, "", "Shared memory")
	memFields := []statefmt.Field{
		{Key: "path", Value: statefmt.FileStatusLine(sharedPath)},
	}
	if v := os.Getenv("ELEGANT_GIT_STATE_FILE"); v != "" {
		memFields = append(memFields, statefmt.Field{Key: "override", Value: "ELEGANT_GIT_STATE_FILE=" + v})
	}

	s, err := shared.Load()
	if err != nil {
		return err
	}
	memFields = append(memFields,
		statefmt.Field{Key: "workspaces", Value: fmt.Sprintf("%d", len(s.Workspaces))},
		statefmt.Field{Key: "repositories", Value: fmt.Sprintf("%d", statefmt.ReposWithWorkspace(s))},
	)
	statefmt.PrintFields(w, "  ", memFields)
	fmt.Fprintln(w)

	statefmt.PrintHeading(w, "", "Global git identity")
	statefmt.PrintGitIdentity(w, "  ", git.ConfigGlobalGet)

	if acquired := shared.Acquired(s); acquired != "" {
		fmt.Fprintf(w, "  elegant-git.acquired: %s\n", acquired)
	} else if acquired := git.ConfigGlobalGet("elegant-git.acquired"); acquired != "" {
		fmt.Fprintf(w, "  elegant-git.acquired: %s (legacy git config; migrates automatically)\n", acquired)
	} else {
		fmt.Fprintln(w, "  elegant-git.acquired: (not set; run self configure)")
	}

	statefmt.PrintFurtherSteps(w, []statefmt.Step{
		{Command: "eg workspace list all", Comment: "the identities you commit with"},
		{Command: "eg repo list all", Comment: "the repositories Elegant Git looks after"},
	})
	return nil
}
