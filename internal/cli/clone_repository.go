package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newCloneRepositoryCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:                spec.name,
		Short:              spec.purpose,
		Long:               spec.purpose,
		DisableFlagParsing: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithWorkflows(cmd, func() error {
				return cloneRepositoryRun(args)
			})
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func cloneRepositoryRun(args []string) error {
	requireArgs(args, "There are no arguments!")
	if err := git.Verbose(append([]string{"clone"}, args...)...); err != nil {
		return err
	}
	location := cloneTargetDir(args)
	text.InfoText(fmt.Sprintf("The repository was cloned into '%s' directory.", location))
	if err := os.Chdir(location); err != nil {
		return err
	}
	return runAcquireRepositoryHooks(acquireRepositoryRun)
}

func cloneTargetDir(args []string) string {
	location := args[len(args)-1]
	if strings.HasSuffix(location, ".git") {
		base := filepath.Base(location)
		location = strings.TrimSuffix(base, ".git")
	}
	return location
}
