package doctor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/hooks"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/memory/repoid"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/runtime"
	"github.com/extsoft/elegant-git/internal/text"
)

// CurrentRepo diagnoses the repository at the current working directory: registry
// and identity drift, leftover local install markers and branch keys, personal
// hooks under .git/.workflows/, and repo-tracked hooks under .workflows/.
func CurrentRepo(s *shared.State, p prompt.Prompter, w io.Writer) ([]Finding, error) {
	if w == nil {
		w = io.Discard
	}
	if _, err := memrepo.GitDir(); err != nil {
		return nil, fmt.Errorf("not a git repository")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cwd, err = filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}

	repoID, _ := repoid.ReadLocal()
	var out []Finding
	out = append(out, diagnoseRepoLinkage(s, cwd, repoID, p)...)
	if repoID == "" {
		if id := findRepoByPath(s, cwd); id != "" {
			repoID = id
		}
	}
	out = append(out, diagnoseRepoIdentity(s, repoID)...)
	out = append(out, diagnoseRepoLegacy(s)...)
	out = append(out, diagnoseLegacyHooks(w)...)
	return out, nil
}

func diagnoseRepoLinkage(s *shared.State, cwd, repoID string, p prompt.Prompter) []Finding {
	var out []Finding

	if repoID == "" {
		if matchID := findRepoByPath(s, cwd); matchID != "" {
			out = append(out, Finding{
				Problem: fmt.Sprintf("%s is unset but registry entry %s matches this path", repoid.Key, matchID),
				Repair:  "stamp the registry id into local git config",
				Apply: func() error {
					return repoid.StampLocal(matchID)
				},
			})
			repoID = matchID
		} else {
			out = append(out, Finding{
				Problem: fmt.Sprintf("%s is unset and no registry entry matches this path", repoid.Key),
				Repair:  "run `eg repo configure <workspace>`",
			})
			return out
		}
	}

	reg, err := shared.GetRepo(s, repoID)
	if err != nil {
		out = append(out, Finding{
			Problem: fmt.Sprintf("%s %s has no registry entry", repoid.Key, repoID),
			Repair:  "re-add this repository to shared memory",
			Apply: func() error {
				return readdCurrentRepo(s, cwd, repoID, p)
			},
		})
		return out
	}

	if reg.CurrentPath != cwd {
		out = append(out, Finding{
			Problem: fmt.Sprintf("registry path %s differs from cwd %s", reg.CurrentPath, cwd),
			Repair:  "update registry current_path (old path moves to path_history)",
			Apply: func() error {
				_, err := shared.UpdateRepoPath(s, repoID, cwd)
				return err
			},
		})
	}

	if reg.WorkspaceID == "" {
		out = append(out, Finding{
			Problem: "registry workspace_id is empty",
			Repair:  "pick a workspace and link this repository",
			Apply: func() error {
				return linkCurrentRepoWorkspace(s, repoID, p)
			},
		})
	} else if _, err := shared.GetWorkspace(s, reg.WorkspaceID); err != nil {
		out = append(out, Finding{
			Problem: fmt.Sprintf("registry workspace_id %s is missing", reg.WorkspaceID),
			Repair:  "pick a workspace and link this repository",
			Apply: func() error {
				return linkCurrentRepoWorkspace(s, repoID, p)
			},
		})
	}

	gitDir, err := memrepo.GitDir()
	if err != nil {
		return out
	}
	perRepo, err := memrepo.Load(gitDir)
	if err != nil {
		return out
	}
	if perRepo.RepoID != "" && perRepo.RepoID != repoID {
		out = append(out, Finding{
			Problem: fmt.Sprintf("per-repo memory repo_id %s differs from %s", perRepo.RepoID, repoID),
			Repair:  "rewrite per-repo memory repo_id",
			Apply: func() error {
				perRepo.RepoID = repoID
				return memrepo.Save(gitDir, perRepo)
			},
		})
	}
	if reg.WorkspaceID != "" && perRepo.WorkspaceID != "" && perRepo.WorkspaceID != reg.WorkspaceID {
		wsID := reg.WorkspaceID
		out = append(out, Finding{
			Problem: "per-repo memory workspace_id differs from registry",
			Repair:  "rewrite per-repo memory workspace_id",
			Apply: func() error {
				perRepo.WorkspaceID = wsID
				if perRepo.RepoID == "" {
					perRepo.RepoID = repoID
				}
				return memrepo.Save(gitDir, perRepo)
			},
		})
	}
	return out
}

func diagnoseRepoIdentity(s *shared.State, repoID string) []Finding {
	if repoID == "" {
		return nil
	}
	reg, err := shared.GetRepo(s, repoID)
	if err != nil || reg.WorkspaceID == "" {
		return nil
	}
	ws, err := shared.GetWorkspace(s, reg.WorkspaceID)
	if err != nil {
		return nil
	}
	if !identityDiffers(ws) {
		return nil
	}
	return []Finding{{
		Problem: "local git identity differs from the linked workspace",
		Repair:  "re-apply workspace identity with force",
		Apply: func() error {
			return shared.ApplyWorkspace(ws, prompt.NewNonInteractive(), &shared.Apply{Force: true})
		},
	}}
}

func identityDiffers(ws *shared.Workspace) bool {
	if ws.UserName != "" && git.ConfigLocalGet("user.name") != ws.UserName {
		return true
	}
	if ws.UserEmail != "" && git.ConfigLocalGet("user.email") != ws.UserEmail {
		return true
	}
	if ws.SigningKey != "" && git.ConfigLocalGet("user.signingkey") != ws.SigningKey {
		return true
	}
	if ws.GPGProgram != "" && git.ConfigLocalGet("gpg.program") != ws.GPGProgram {
		return true
	}
	if ws.Editor != "" && git.ConfigLocalGet("core.editor") != ws.Editor {
		return true
	}
	return false
}

func diagnoseRepoLegacy(s *shared.State) []Finding {
	var out []Finding

	def, prot := memrepo.ReadLegacyElegantGitSettings()
	if def != "" || len(prot) > 0 {
		out = append(out, Finding{
			Problem: "legacy elegant-git.default-branch / protected-branches still in local config",
			Repair:  "move values into per-repo memory and unset the keys",
			Apply: func() error {
				gitDir, err := memrepo.GitDir()
				if err != nil {
					return err
				}
				perRepo, err := memrepo.Load(gitDir)
				if err != nil {
					return err
				}
				if def != "" {
					perRepo.DefaultBranch = def
				}
				if len(prot) > 0 {
					perRepo.ProtectedBranches = prot
				}
				if err := memrepo.Save(gitDir, perRepo); err != nil {
					return err
				}
				return memrepo.UnsetLegacyElegantGitKeys()
			},
		})
	}

	if config.IsGitAcquired() && hasRedundantLocalInstall() {
		out = append(out, Finding{
			Problem: "global Elegant Git is acquired but local elegant aliases or acquired marker remain",
			Repair:  "remove redundant local install markers and aliases",
			Apply: func() error {
				return config.CleanupRedundantLocalInstall(false)
			},
		})
	}
	return out
}

func diagnoseLegacyHooks(w io.Writer) []Finding {
	ws := runtime.DefaultRepoLayout()
	var out []Finding
	if hooks.HasLegacy(ws, true) {
		out = append(out, Finding{
			Problem: "personal hooks still live under .git/.workflows/",
			Repair:  "move them to .git/.config/elegant-git/hooks/",
			Apply: func() error {
				_, _, err := hooks.Migrate(ws, true, false, w)
				return err
			},
		})
	}
	if hooks.HasLegacy(ws, false) {
		out = append(out, Finding{
			Problem: "repo-tracked hooks still live under .workflows/",
			Repair:  "move them to .config/elegant-git/hooks/ and commit the result",
			Apply: func() error {
				newPaths, oldPaths, err := hooks.Migrate(ws, false, false, w)
				if err != nil {
					return err
				}
				text.SuggestGitAddCommitTo(w, newPaths, oldPaths, "Migrate Elegant Git hooks")
				return nil
			},
		})
	}
	return out
}

func findRepoByPath(s *shared.State, absPath string) string {
	for id, repo := range s.Repositories {
		if repo != nil && repo.CurrentPath == absPath {
			return id
		}
	}
	return ""
}

func readdCurrentRepo(s *shared.State, cwd, repoID string, p prompt.Prompter) error {
	workspaceID := ""
	if gitDir, err := memrepo.GitDir(); err == nil {
		if perRepo, err := memrepo.Load(gitDir); err == nil && perRepo.WorkspaceID != "" {
			if _, err := shared.GetWorkspace(s, perRepo.WorkspaceID); err == nil {
				workspaceID = perRepo.WorkspaceID
			}
		}
	}
	if workspaceID == "" {
		id, err := pickWorkspace(s, p)
		if err != nil {
			return err
		}
		workspaceID = id
	}
	origin := strings.TrimSpace(git.OutputOK("config", "--get", "remote.origin.url"))
	return shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: repoID, Name: filepath.Base(cwd), WorkspaceID: workspaceID, CurrentPath: cwd, OriginURL: origin,
	})
}

func linkCurrentRepoWorkspace(s *shared.State, repoID string, p prompt.Prompter) error {
	workspaceID, err := pickWorkspace(s, p)
	if err != nil {
		return err
	}
	return shared.Relink(s, repoID, workspaceID)
}

func pickWorkspace(s *shared.State, p prompt.Prompter) (string, error) {
	choices := make([]prompt.Choice, 0, len(s.Workspaces))
	for _, ws := range shared.ListWorkspaces(s) {
		if ws == nil {
			continue
		}
		choices = append(choices, prompt.Choice{
			Value:       ws.Name,
			Description: ws.UserName + " <" + ws.UserEmail + ">",
		})
	}
	sort.Slice(choices, func(i, j int) bool { return choices[i].Value < choices[j].Value })
	if len(choices) == 0 {
		return "", fmt.Errorf("no workspaces available; create one with workspace new")
	}
	if prompt.NonInteractive(p) {
		return "", fmt.Errorf("workspace selection requires interactive mode")
	}
	picked, err := p.Pick("Workspace", choices, "")
	if err != nil {
		return "", err
	}
	id, _, err := shared.GetWorkspaceByName(s, picked)
	return id, err
}

func hasRedundantLocalInstall() bool {
	if git.ConfigLocalGet(config.AcquiredKey) != "" {
		return true
	}
	out := git.OutputOK("config", "--local", "--get-regexp", `^alias\.`)
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value := strings.Join(fields[1:], " ")
		if config.IsRemovableAliasValue(value) {
			return true
		}
	}
	return false
}
