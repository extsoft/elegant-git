package cli

import (
	"strings"

	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/state"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newPruneRepositoryCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWithWorkflows(cmd, pruneRepositoryRun)
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func pruneRepositoryRun() error {
	defaultBranch := config.DefaultBranch()
	if err := git.Verbose("checkout", defaultBranch); err != nil {
		return err
	}
	if state.IsThereUpstreamFor(defaultBranch) {
		if err := git.Verbose("fetch", "--all"); err != nil {
			text.InfoText("As the remotes can't be fetched, the current local version is used.")
		} else {
			if err := git.Verbose("rebase"); err != nil {
				return err
			}
		}
	}
	branches := strings.Split(strings.TrimSpace(git.OutputOK("for-each-ref", "--format", "%(refname:short)", "refs/heads")), "\n")
	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		if branch == "" || config.IsBranchProtected(branch) {
			continue
		}
		merge, _ := git.Output("config", "--get", "branch."+branch+".merge")
		if strings.TrimSpace(merge) != "" {
			if state.IsThereUpstreamFor(branch) {
				continue
			}
		} else {
			base, _ := git.Output("merge-base", defaultBranch, branch)
			tip, _ := git.Output("rev-parse", branch)
			if strings.TrimSpace(base) != strings.TrimSpace(tip) {
				continue
			}
		}
		if err := git.Verbose("branch", "--delete", "--force", branch); err != nil {
			return err
		}
	}
	return nil
}
