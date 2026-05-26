package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	memrepo "github.com/bees-hive/elegant-git/internal/memory/repo"
	"github.com/bees-hive/elegant-git/internal/memory/repoid"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

const (
	pickerCreateNew    = "[Create new]"
	pickerImportGitCfg = "[Import from this repository]"
)

func configureWithMemory(cmd *cobra.Command, profileFlag string) error {
	p := prompt.FromContext(cmd.Context())
	repoID, err := repoid.EnsureLocal()
	if err != nil {
		return err
	}
	_ = shared.TouchCurrentRepo()
	gitDir, err := memrepo.GitDir()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cwd, _ = filepath.Abs(cwd)

	sharedState, err := shared.Load()
	if err != nil {
		return err
	}
	perRepo, err := memrepo.Load(gitDir)
	if err != nil {
		return err
	}
	perRepo.RepoID = repoID

	profileID, prof, err := resolveProfile(cmd, sharedState, profileFlag, p)
	if err != nil {
		return err
	}

	repoName := filepath.Base(cwd)
	origin := strings.TrimSpace(git.OutputOK("config", "--get", "remote.origin.url"))
	if err := shared.UpsertRepo(sharedState, shared.UpsertRepoInput{
		ID: repoID, Name: repoName, ProfileID: profileID, CurrentPath: cwd, OriginURL: origin,
	}); err != nil {
		return err
	}
	if r, _ := shared.GetRepo(sharedState, repoID); r != nil {
		_ = shared.RecordPath(sharedState, repoID, cwd)
	}

	perRepo.ProfileID = profileID
	if err := shared.ApplyProfile(prof, p, &shared.Apply{Force: true}); err != nil {
		return err
	}

	if err := configureElegantRepoSettings(perRepo, p); err != nil {
		return err
	}
	if err := memrepo.UnsetLegacyElegantGitKeys(); err != nil {
		return err
	}
	if err := memrepo.Save(gitDir, perRepo); err != nil {
		return err
	}
	_ = shared.Validate(sharedState)
	return shared.Save(sharedState)
}

func resolveProfile(cmd *cobra.Command, s *shared.State, profileFlag string, p prompt.Prompter) (string, *shared.Profile, error) {
	if profileFlag != "" {
		id, prof, err := shared.GetProfileByName(s, profileFlag)
		if err != nil {
			return "", nil, fmt.Errorf("profile %q: %w", profileFlag, err)
		}
		return id, prof, nil
	}
	if prompt.NonInteractive(p) {
		return "", nil, fmt.Errorf("repo configure: --profile is required in non-interactive mode")
	}
	options := []string{}
	for _, prof := range shared.ListProfiles(s) {
		if prof != nil {
			options = append(options, prof.Name)
		}
	}
	options = append(options, pickerCreateNew)
	if hasLocalUserConfig() {
		options = append(options, pickerImportGitCfg)
	}
	idx, err := p.Choose("Select a profile", options)
	if err != nil {
		return "", nil, err
	}
	choice := options[idx]
	switch choice {
	case pickerCreateNew:
		return createProfileInteractive(p, s)
	case pickerImportGitCfg:
		return importProfileFromGitConfig(p, s)
	default:
		id, prof, err := shared.GetProfileByName(s, choice)
		return id, prof, err
	}
}

func hasLocalUserConfig() bool {
	n := git.ConfigLocalGet("user.name")
	e := git.ConfigLocalGet("user.email")
	return n != "" && e != ""
}

func importProfileFromGitConfig(p prompt.Prompter, s *shared.State) (string, *shared.Profile, error) {
	userName := git.ConfigLocalGet("user.name")
	userEmail := git.ConfigLocalGet("user.email")
	signingKey := git.ConfigLocalGet("user.signingkey")
	editor := git.ConfigLocalGet("core.editor")
	gpgProgram := git.ConfigLocalGet("gpg.program")
	profName, err := p.EditOrAccept("Profile name", defaultProfileName(userEmail))
	if err != nil {
		return "", nil, err
	}
	id, err := shared.CreateProfile(s, shared.CreateProfileInput{
		Name: profName, UserName: userName, UserEmail: userEmail,
		SigningKey: signingKey, Editor: editor, GPGProgram: gpgProgram,
	})
	if err != nil {
		return "", nil, err
	}
	return id, s.Profiles[id], nil
}

func createProfileInteractive(p prompt.Prompter, s *shared.State) (string, *shared.Profile, error) {
	emailHint := git.ConfigEffectiveLocal("user.email")
	name, err := p.EditOrAccept("Profile name", defaultProfileName(emailHint))
	if err != nil {
		return "", nil, err
	}
	userName, err := prompt.Skippable(p, "Git user.name", "", git.ConfigEffectiveLocal("user.name"))
	if err != nil {
		return "", nil, err
	}
	userEmail, err := prompt.Skippable(p, "Git user.email", "", git.ConfigEffectiveLocal("user.email"))
	if err != nil {
		return "", nil, err
	}
	signingKey, err := p.EditOrAccept("Signing key (empty to skip)", git.ConfigEffectiveLocal("user.signingkey"))
	if err != nil {
		return "", nil, err
	}
	gpgProgram, err := p.EditOrAccept("GPG program (empty to skip)", git.ConfigEffectiveLocal("gpg.program"))
	if err != nil {
		return "", nil, err
	}
	editor, err := p.EditOrAccept("Editor command (empty to skip)", git.ConfigEffectiveLocal("core.editor"))
	if err != nil {
		return "", nil, err
	}
	id, err := shared.CreateProfile(s, shared.CreateProfileInput{
		Name: name, UserName: userName, UserEmail: userEmail,
		SigningKey: signingKey, Editor: editor, GPGProgram: gpgProgram,
	})
	if err != nil {
		return "", nil, err
	}
	return id, s.Profiles[id], nil
}

func defaultProfileName(email string) string {
	if i := strings.Index(email, "@"); i > 0 {
		return email[:i]
	}
	return email
}

func configureElegantRepoSettings(perRepo *memrepo.State, p prompt.Prompter) error {
	def := perRepo.DefaultBranch
	if def == "" {
		legacyDef, _ := memrepo.ReadLegacyElegantGitSettings()
		def = legacyDef
	}
	if def == "" {
		def = config.DefaultBranchDefault
	}
	if prompt.NonInteractive(p) {
		perRepo.DefaultBranch = def
	} else {
		v, err := p.String("What is the default branch?", def)
		if err != nil {
			return err
		}
		if v != "" {
			perRepo.DefaultBranch = v
		} else {
			perRepo.DefaultBranch = def
		}
	}
	prot := perRepo.ProtectedBranches
	if len(prot) == 0 {
		_, legacyProt := memrepo.ReadLegacyElegantGitSettings()
		prot = legacyProt
	}
	if len(prot) == 0 {
		prot = []string{config.ProtectedBranchesDef}
	}
	protStr := strings.Join(prot, " ")
	if prompt.NonInteractive(p) {
		perRepo.ProtectedBranches = prot
	} else {
		v, err := p.String("What are protected branches (split with space)?", protStr)
		if err != nil {
			return err
		}
		if v != "" {
			perRepo.ProtectedBranches = strings.Fields(v)
		} else {
			perRepo.ProtectedBranches = prot
		}
	}
	return nil
}
