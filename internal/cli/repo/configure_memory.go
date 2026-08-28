package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bees-hive/elegant-git/internal/cli/sources"
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	memrepo "github.com/bees-hive/elegant-git/internal/memory/repo"
	"github.com/bees-hive/elegant-git/internal/memory/repoid"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

func configureWithMemory(cmd *cobra.Command, workspaceName string) error {
	p := prompt.FromContext(cmd.Context())
	repoID, err := repoid.EnsureLocal()
	if err != nil {
		return err
	}
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

	workspaceID, ws, err := resolveWorkspace(sharedState, workspaceName, p)
	if err != nil {
		return err
	}

	repoName := filepath.Base(cwd)
	origin := strings.TrimSpace(git.OutputOK("config", "--get", "remote.origin.url"))
	if err := shared.UpsertRepo(sharedState, shared.UpsertRepoInput{
		ID: repoID, Name: repoName, WorkspaceID: workspaceID, CurrentPath: cwd, OriginURL: origin,
	}); err != nil {
		return err
	}

	perRepo.WorkspaceID = workspaceID
	if err := shared.ApplyWorkspace(ws, p, &shared.Apply{Force: true}); err != nil {
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

func resolveWorkspace(s *shared.State, workspaceName string, p prompt.Prompter) (string, *shared.Workspace, error) {
	if workspaceName == sources.WorkspaceCreateNew {
		if prompt.NonInteractive(p) {
			return "", nil, fmt.Errorf("workspace creation requires interactive mode")
		}
		return createWorkspaceInteractive(p, s)
	}
	id, ws, err := shared.GetWorkspaceByName(s, workspaceName)
	if err != nil {
		return "", nil, fmt.Errorf("workspace %q: %w", workspaceName, err)
	}
	return id, ws, nil
}

func createWorkspaceInteractive(p prompt.Prompter, s *shared.State) (string, *shared.Workspace, error) {
	emailHint := git.ConfigEffectiveLocal("user.email")
	name, err := p.EditOrAccept("Workspace name", defaultWorkspaceName(emailHint))
	if err != nil {
		return "", nil, err
	}
	userName, err := p.EditOrAccept("Git user.name", git.ConfigEffectiveLocal("user.name"))
	if err != nil {
		return "", nil, err
	}
	userEmail, err := p.EditOrAccept("Git user.email", git.ConfigEffectiveLocal("user.email"))
	if err != nil {
		return "", nil, err
	}
	signingKey, err := p.Optional("Signing key", git.ConfigEffectiveLocal("user.signingkey"))
	if err != nil {
		return "", nil, err
	}
	gpgProgram, err := p.Optional("GPG program", git.ConfigEffectiveLocal("gpg.program"))
	if err != nil {
		return "", nil, err
	}
	editor, err := p.Optional("Editor command", git.ConfigEffectiveLocal("core.editor"))
	if err != nil {
		return "", nil, err
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: name, UserName: userName, UserEmail: userEmail,
		SigningKey: signingKey, Editor: editor, GPGProgram: gpgProgram,
	})
	if err != nil {
		return "", nil, err
	}
	return id, s.Workspaces[id], nil
}

func defaultWorkspaceName(email string) string {
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
		v, err := p.EditOrAccept("Default branch", def)
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
		v, err := p.EditOrAccept("Protected branches", protStr)
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
