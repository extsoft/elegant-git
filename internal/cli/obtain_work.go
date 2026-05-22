package cli

import (
	"fmt"
	"strings"

	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newObtainWorkCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithWorkflows(cmd, func() error {
				return obtainWorkRun(args)
			})
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func obtainWorkRun(args []string) error {
	requireArgs(args, "Please provide a branch name or its part.")
	return obtainWorkLogic(args[0], argAt(args, 1))
}

func obtainWorkLogic(pattern, localBranch string) error {
	if err := git.Verbose("fetch", "--all"); err != nil {
		return err
	}
	remotes := strings.Split(git.OutputOK("for-each-ref", "--format=%(refname:short)", "refs/remotes"), "\n")
	var matches []string
	for _, ref := range remotes {
		ref = strings.TrimSpace(ref)
		if ref != "" && strings.Contains(ref, pattern) {
			matches = append(matches, ref)
		}
	}
	if len(matches) > 1 {
		text.InfoText("The following branches are found:")
		for _, b := range matches {
			text.InfoText(" - " + b)
		}
		exitWorkflowError("Please re-run the command with concrete branch name from the list above!")
	}
	if len(matches) == 0 {
		exitWorkflowError(fmt.Sprintf("There is no branch that matches the '%s' pattern.", pattern))
	}
	remote := matches[0]
	local := localBranch
	if local == "" {
		local = branchFromRemoteBranch(remote)
	}
	return git.Verbose("checkout", "-B", local, remote)
}

func argAt(args []string, i int) string {
	if len(args) > i {
		return args[i]
	}
	return ""
}
