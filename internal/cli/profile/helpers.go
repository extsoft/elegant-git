package profile

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

func defaultProfileName(email string) string {
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

func offerApplyToCurrentRepo(cmd *cobra.Command, s *shared.State, profileID string, prof *shared.Profile) error {
	p := prompt.FromContext(cmd.Context())
	if prompt.NonInteractive(p) {
		return nil
	}
	if _, err := memrepo.GitDir(); err != nil {
		return nil
	}
	ok, err := p.Confirm(fmt.Sprintf(`Apply profile "%s" to current repository?`, prof.Name))
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
	if existing, _ := shared.GetRepo(s, repoID); existing != nil && existing.ProfileID != "" && existing.ProfileID != profileID {
		other, _ := shared.GetProfile(s, existing.ProfileID)
		otherName := existing.ProfileID
		if other != nil {
			otherName = other.Name
		}
		ok, err := p.Confirm(fmt.Sprintf(`Override existing profile "%s"?`, otherName))
		if err != nil || !ok {
			return err
		}
	}
	repoName := filepath.Base(cwd)
	origin := strings.TrimSpace(git.OutputOK("config", "--get", "remote.origin.url"))
	if err := shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: repoID, Name: repoName, ProfileID: profileID, CurrentPath: cwd, OriginURL: origin,
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
	perRepo.ProfileID = profileID
	if err := shared.ApplyProfile(prof, p, nil); err != nil {
		return err
	}
	return memrepo.Save(gitDir, perRepo)
}
