package repo

import (
	"strings"

	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/memory/repo"
	"github.com/bees-hive/elegant-git/internal/state"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var pruneID = cmdid.ID{Command: "repo", Action: "prune"}

func newPruneCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "prune",
		Short: "Removes useless local branches",
		Long:  "Deletes merged or obsolete local branches.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cliruntime.RunWithWorkflows(cmd, pruneID, func() error {
				return pruneRun()
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func pruneRun() error {
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
		if err := repo.ClearBranchSourceFromCWD(branch); err != nil {
			return err
		}
	}
	return nil
}
