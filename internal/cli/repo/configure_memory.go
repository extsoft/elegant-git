package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bees-hive/elegant-git/internal/cli/sources"
	"github.com/bees-hive/elegant-git/internal/cli/workspace"
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/giturl"
	memrepo "github.com/bees-hive/elegant-git/internal/memory/repo"
	"github.com/bees-hive/elegant-git/internal/memory/repoid"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

const showAllWorkspaces = "[Show all workspaces]"

func configureWithMemory(cmd *cobra.Command, workspaceName string) (bool, error) {
	p := prompt.FromContext(cmd.Context())
	sharedState, err := shared.Load()
	if err != nil {
		return false, err
	}
	origin := strings.TrimSpace(git.OutputOK("config", "--get", "remote.origin.url"))
	workspaceID, ws, err := resolveWorkspace(sharedState, workspaceName, origin, p)
	if err != nil {
		return false, err
	}
	if workspaceID == "" || ws == nil {
		text.InfoText("No workspace assigned.")
		return false, nil
	}

	repoID, err := repoid.EnsureLocal()
	if err != nil {
		return false, err
	}
	gitDir, err := memrepo.GitDir()
	if err != nil {
		return false, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return false, err
	}
	cwd, _ = filepath.Abs(cwd)

	perRepo, err := memrepo.Load(gitDir)
	if err != nil {
		return false, err
	}
	perRepo.RepoID = repoID

	repoName := filepath.Base(cwd)
	if err := shared.UpsertRepo(sharedState, shared.UpsertRepoInput{
		ID: repoID, Name: repoName, WorkspaceID: workspaceID, CurrentPath: cwd, OriginURL: origin,
	}); err != nil {
		return false, err
	}

	perRepo.WorkspaceID = workspaceID
	if err := shared.ApplyWorkspace(ws, p, &shared.Apply{Force: true}); err != nil {
		return false, err
	}

	if err := configureElegantRepoSettings(perRepo, p); err != nil {
		return false, err
	}
	if err := memrepo.UnsetLegacyElegantGitKeys(); err != nil {
		return false, err
	}
	if err := memrepo.Save(gitDir, perRepo); err != nil {
		return false, err
	}
	if err := workspace.CaptureNamespace(sharedState, workspaceID, ws, origin, p); err != nil {
		return false, err
	}
	_ = shared.Validate(sharedState)
	return true, shared.Save(sharedState)
}

func resolveWorkspace(s *shared.State, workspaceName, origin string, p prompt.Prompter) (string, *shared.Workspace, error) {
	if workspaceName == "" {
		return resolveWorkspaceFromNamespace(s, origin, p)
	}
	if workspaceName == sources.WorkspaceCreateNew {
		if prompt.NonInteractive(p) {
			return "", nil, fmt.Errorf("workspace creation requires interactive mode")
		}
		ns, _ := giturl.Namespace(origin)
		var seed []string
		if ns != "" {
			seed = []string{ns}
		}
		return createWorkspaceInteractive(p, s, seed)
	}
	id, ws, err := shared.GetWorkspaceByName(s, workspaceName)
	if err != nil {
		return "", nil, fmt.Errorf("workspace %q: %w", workspaceName, err)
	}
	return id, ws, nil
}

func resolveWorkspaceFromNamespace(s *shared.State, origin string, p prompt.Prompter) (string, *shared.Workspace, error) {
	ns, ok := giturl.Namespace(origin)
	var matches []string
	if ok {
		matches = shared.FindWorkspacesByNamespace(s, ns)
	}

	if prompt.NonInteractive(p) {
		if len(matches) == 1 {
			ws, err := shared.GetWorkspace(s, matches[0])
			if err != nil {
				return "", nil, err
			}
			return matches[0], ws, nil
		}
		return "", nil, nil
	}

	switch len(matches) {
	case 1:
		ws, err := shared.GetWorkspace(s, matches[0])
		if err != nil {
			return "", nil, err
		}
		ok, err := p.Confirm(fmt.Sprintf("Use workspace %q? (matched namespace %s)", ws.Name, ns), true)
		if err != nil {
			return "", nil, err
		}
		if ok {
			return matches[0], ws, nil
		}
	case 0:
		// fall through to full picker
	default:
		id, ws, err := pickAmongMatches(s, matches, ns, p)
		if err != nil {
			return "", nil, err
		}
		if id != "" {
			return id, ws, nil
		}
		// user chose show-all; fall through
	}
	return pickWorkspaceOrCreate(s, ns, p)
}

func pickAmongMatches(s *shared.State, ids []string, ns string, p prompt.Prompter) (string, *shared.Workspace, error) {
	choices := make([]prompt.Choice, 0, len(ids)+1)
	for _, id := range ids {
		ws, err := shared.GetWorkspace(s, id)
		if err != nil {
			continue
		}
		choices = append(choices, prompt.Choice{
			Value:       ws.Name,
			Description: ws.UserName + " <" + ws.UserEmail + ">",
		})
	}
	choices = append(choices, prompt.Choice{Value: showAllWorkspaces, Description: "Show all workspaces"})
	label := "Workspace"
	if ns != "" {
		label = fmt.Sprintf("Workspace (namespace %s)", ns)
	}
	name, err := p.Pick(label, choices, "")
	if err != nil {
		return "", nil, err
	}
	if name == showAllWorkspaces {
		return "", nil, nil
	}
	return shared.GetWorkspaceByName(s, name)
}

func pickWorkspaceOrCreate(s *shared.State, ns string, p prompt.Prompter) (string, *shared.Workspace, error) {
	choices := make([]prompt.Choice, 0, len(s.Workspaces)+1)
	for _, ws := range shared.ListWorkspaces(s) {
		if ws == nil {
			continue
		}
		choices = append(choices, prompt.Choice{
			Value:       ws.Name,
			Description: ws.UserName + " <" + ws.UserEmail + ">",
		})
	}
	choices = append(choices, prompt.Choice{Value: sources.WorkspaceCreateNew, Description: "Create a new workspace"})
	label := "Workspace"
	if ns != "" {
		label = fmt.Sprintf("Workspace (no match for namespace %s)", ns)
	}
	name, err := p.Pick(label, choices, "")
	if err != nil {
		if err == prompt.ErrUserCancelled {
			return "", nil, fmt.Errorf("workspace is required")
		}
		return "", nil, err
	}
	if name == sources.WorkspaceCreateNew {
		var seed []string
		if ns != "" {
			seed = []string{ns}
		}
		return createWorkspaceInteractive(p, s, seed)
	}
	return shared.GetWorkspaceByName(s, name)
}

func createWorkspaceInteractive(p prompt.Prompter, s *shared.State, seedNamespaces []string) (string, *shared.Workspace, error) {
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
		Namespaces: seedNamespaces,
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
