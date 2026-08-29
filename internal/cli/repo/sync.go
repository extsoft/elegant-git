package repo

import (
	"fmt"
	"os"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/memory/repoid"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var syncID = cmdid.ID{Command: "repo", Action: "sync"}

func newSyncCommand() *cobra.Command {
	var all bool
	c := &cobra.Command{
		Use:   "sync",
		Short: "Re-apply workspace settings to repositories",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cliruntime.RunWithWorkflows(cmd, syncID, func() error {
				return syncRun(cmd, all)
			})
		},
	}
	c.Flags().BoolVar(&all, "all", false, "sync every managed repository")
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func syncRun(cmd *cobra.Command, all bool) error {
	p := prompt.FromContext(cmd.Context())
	s, err := shared.Load()
	if err != nil {
		return err
	}
	_ = shared.TouchCurrentRepo()
	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()

	var targets []string
	if all {
		for id := range shared.ListRepos(s) {
			targets = append(targets, id)
		}
	} else {
		id, err := repoid.ReadLocal()
		if err != nil || id == "" {
			return fmt.Errorf("not a configured elegant-git repository; run repo configure first")
		}
		targets = []string{id}
	}

	applyAll := prompt.NonInteractive(p)
	skipRemaining := false
	missingHint := ""
	for _, repoID := range targets {
		if skipRemaining {
			continue
		}
		repo, err := shared.GetRepo(s, repoID)
		if err != nil {
			continue
		}
		prof, err := shared.GetWorkspace(s, repo.WorkspaceID)
		if err != nil {
			if repo.WorkspaceID == "" {
				text.InfoText("repo " + repo.Name + ": no workspace linked; skip")
			} else {
				text.ErrorText("repo " + repo.Name + ": workspace missing")
			}
			continue
		}
		apply := &shared.Apply{Force: applyAll}
		if !applyAll {
			dec, err := p.BatchChoice(fmt.Sprintf("Apply %s?", repo.Name), "no")
			if err != nil {
				return err
			}
			switch dec {
			case prompt.BatchConfirm:
			case prompt.BatchApplyAll:
				apply.Force = true
				applyAll = true
			case prompt.BatchSkip:
				skipRemaining = true
				continue
			default:
				continue
			}
		}
		if err := os.Chdir(repo.CurrentPath); err != nil {
			text.ErrorText("repo " + repo.Name + ": path missing: " + repo.CurrentPath)
			if missingHint == "" {
				missingHint = prof.Name
			}
			continue
		}
		if err := shared.ApplyWorkspace(prof, p, apply); err != nil {
			return err
		}
		text.InfoText("Synced " + repo.Name)
	}
	if missingHint != "" {
		text.InfoText("Run `git elegant workspace doctor " + missingHint + "` to repair missing paths from the workspace side, or `cd` into a relocated clone and run `git elegant repo doctor`.")
	}
	return nil
}
