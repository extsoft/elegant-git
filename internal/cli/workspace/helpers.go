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
	_, err = applyToCurrentRepo(cmd, s, workspaceID, ws)
	return err
}

func applyToCurrentRepo(cmd *cobra.Command, s *shared.State, workspaceID string, ws *shared.Workspace) (bool, error) {
	p := prompt.FromContext(cmd.Context())
	repoID, err := repoid.EnsureLocal()
	if err != nil {
		return false, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return false, err
	}
	cwd, _ = filepath.Abs(cwd)
	if existing, _ := shared.GetRepo(s, repoID); existing != nil && existing.WorkspaceID != "" && existing.WorkspaceID != workspaceID {
		other, _ := shared.GetWorkspace(s, existing.WorkspaceID)
		otherName := existing.WorkspaceID
		if other != nil {
			otherName = other.Name
		}
		ok, err := p.Confirm(fmt.Sprintf(`Override existing workspace "%s"?`, otherName), false)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	repoName := filepath.Base(cwd)
	origin := strings.TrimSpace(git.OutputOK("config", "--get", "remote.origin.url"))
	// Apply identity before registry mutation so a failed apply does not dirty shared state.
	if err := shared.ApplyWorkspace(ws, p, &shared.Apply{Force: true}); err != nil {
		return false, err
	}
	if err := shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: repoID, Name: repoName, WorkspaceID: workspaceID, CurrentPath: cwd, OriginURL: origin,
	}); err != nil {
		return false, err
	}
	gitDir, err := memrepo.GitDir()
	if err != nil {
		return false, err
	}
	perRepo, err := memrepo.Load(gitDir)
	if err != nil {
		return false, err
	}
	perRepo.RepoID = repoID
	perRepo.WorkspaceID = workspaceID
	if err := CaptureNamespace(s, workspaceID, ws, origin, p); err != nil {
		return false, err
	}
	if err := memrepo.Save(gitDir, perRepo); err != nil {
		return false, err
	}
	return true, nil
}
