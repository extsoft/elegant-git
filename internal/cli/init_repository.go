package cli

import (
	"os"

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

func newInitRepositoryCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWithWorkflows(cmd, initRepositoryRun)
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func initRepositoryRun() error {
	if err := git.Verbose("init"); err != nil {
		return err
	}
	if err := runAcquireRepositoryHooks(acquireRepositoryRun); err != nil {
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
