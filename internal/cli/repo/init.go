package repo

import (
	"os"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/cli/completion"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/spf13/cobra"
)

const initialCommitMessage = `Add initial empty commit

This commit is the first commit in this working tree. It does not have
any changes. However, it simplifies further work at least in the
following cases:
- it's possible to create a branch now
- it's possible to manage the second commit if it requires some
polishing after creation

This commit is created automatically by Elegant Git after the
initialization of a new repository.
`

var initID = cmdid.ID{Command: "repo", Action: "init"}

func newInitCommand() *cobra.Command {
	var workspaceName string
	spec := configureSpec(&workspaceName)
	c := &cobra.Command{
		Use:   "init <workspace>",
		Short: "Initializes a new repository and configures it",
		Long:  "Runs git init, repo configure, and an initial empty commit.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithCompat(cmd, initID, "acquire-repository", func() error {
				if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
					return err
				}
				return initRun(cmd, workspaceName)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func initRun(cmd *cobra.Command, workspace string) error {
	if err := git.Verbose("init"); err != nil {
		return err
	}
	if err := ConfigureRun(cmd, workspace); err != nil {
		return err
	}
	msgFile := "a-message-of-initial-commit"
	if err := os.WriteFile(msgFile, []byte(initialCommitMessage), 0o600); err != nil {
		return err
	}
	defer os.Remove(msgFile)
	if err := git.Verbose("commit", "--allow-empty", "--file", msgFile); err != nil {
		return err
	}
	return git.Verbose("show")
}
