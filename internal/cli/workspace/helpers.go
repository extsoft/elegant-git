package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bees-hive/elegant-git/internal/git"
	memrepo "github.com/bees-hive/elegant-git/internal/memory/repo"
	"github.com/bees-hive/elegant-git/internal/memory/repoid"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

func defaultWorkspaceName(email string) string {
	if i := strings.Index(email, "@"); i > 0 {
		return email[:i]
	}
	return email
}

func suggestField(flagVal, configKey string) string {
	if flagVal != "" {
		return flagVal
	}
	return git.ConfigEffectiveLocal(configKey)
}

func offerApplyToCurrentRepo(cmd *cobra.Command, s *shared.State, workspaceID string, ws *shared.Workspace) error {
	p := prompt.FromContext(cmd.Context())
	if prompt.NonInteractive(p) {
		return nil
	}
	if _, err := memrepo.GitDir(); err != nil {
		return nil
	}
	ok, err := p.Confirm(fmt.Sprintf(`Apply workspace "%s" to current repository?`, ws.Name), true)
	if err != nil || !ok {
		return err
	}
	repoID, err := repoid.EnsureLocal()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cwd, _ = filepath.Abs(cwd)
	if existing, _ := shared.GetRepo(s, repoID); existing != nil && existing.WorkspaceID != "" && existing.WorkspaceID != workspaceID {
		other, _ := shared.GetWorkspace(s, existing.WorkspaceID)
		otherName := existing.WorkspaceID
		if other != nil {
			otherName = other.Name
		}
		ok, err := p.Confirm(fmt.Sprintf(`Override existing workspace "%s"?`, otherName), false)
		if err != nil || !ok {
			return err
		}
	}
	repoName := filepath.Base(cwd)
	origin := strings.TrimSpace(git.OutputOK("config", "--get", "remote.origin.url"))
	if err := shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: repoID, Name: repoName, WorkspaceID: workspaceID, CurrentPath: cwd, OriginURL: origin,
	}); err != nil {
		return err
	}
	gitDir, err := memrepo.GitDir()
	if err != nil {
		return err
	}
	perRepo, err := memrepo.Load(gitDir)
	if err != nil {
		return err
	}
	perRepo.RepoID = repoID
	perRepo.WorkspaceID = workspaceID
	if err := shared.ApplyWorkspace(ws, p, nil); err != nil {
		return err
	}
	return memrepo.Save(gitDir, perRepo)
}
