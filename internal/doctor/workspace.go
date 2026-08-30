package doctor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/memory/repoid"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/text"
)

func applyWorkspaceName(ws *shared.Workspace, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("workspace name is required")
	}
	if shared.IsReservedWorkspaceName(name) {
		return fmt.Errorf("workspace name %q is reserved", name)
	}
	ws.Name = name
	return nil
}

func Workspace(s *shared.State, workspaceID string, p prompt.Prompter) []Finding {
	ws, err := shared.GetWorkspace(s, workspaceID)
	if err != nil {
		return nil
	}
	var out []Finding

	if strings.TrimSpace(ws.Name) == "" {
		out = append(out, Finding{
			Problem: "workspace name is empty",
			Repair:  "ask for a workspace name",
			Apply: func() error {
				name, err := p.EditOrAccept("Workspace name", suggestWorkspaceName())
				if err != nil {
					return err
				}
				return applyWorkspaceName(ws, name)
			},
		})
	} else {
		for otherID, other := range s.Workspaces {
			if otherID == workspaceID || other == nil {
				continue
			}
			if other.Name == ws.Name {
				out = append(out, Finding{
					Problem: fmt.Sprintf("workspace name %q is also used by another workspace", ws.Name),
					Repair:  "ask for a new name for this workspace",
					Apply: func() error {
						name, err := p.EditOrAccept("Workspace name", ws.Name)
						if err != nil {
							return err
						}
						return applyWorkspaceName(ws, name)
					},
				})
				break
			}
		}
	}
	if strings.TrimSpace(ws.UserName) == "" {
		out = append(out, Finding{
			Problem: "user_name is empty",
			Repair:  "ask for Git user.name",
			Apply: func() error {
				v, err := p.EditOrAccept("Git user.name", git.ConfigEffectiveLocal("user.name"))
				if err != nil {
					return err
				}
				if strings.TrimSpace(v) == "" {
					return fmt.Errorf("Git user.name is required")
				}
				ws.UserName = v
				return nil
			},
		})
	}
	if strings.TrimSpace(ws.UserEmail) == "" {
		out = append(out, Finding{
			Problem: "user_email is empty",
			Repair:  "ask for Git user.email",
			Apply: func() error {
				v, err := p.EditOrAccept("Git user.email", git.ConfigEffectiveLocal("user.email"))
				if err != nil {
					return err
				}
				if strings.TrimSpace(v) == "" {
					return fmt.Errorf("Git user.email is required")
				}
				ws.UserEmail = v
				return nil
			},
		})
	}

	linked := append([]string(nil), ws.LinkedRepos...)
	for _, repoID := range linked {
		repoID := repoID
		repo, err := shared.GetRepo(s, repoID)
		if err != nil {
			out = append(out, Finding{
				Problem: fmt.Sprintf("linked_repos holds unknown repository %s", repoID),
				Repair:  "drop that entry from linked_repos",
				Apply: func() error {
					ws.LinkedRepos = filterOut(ws.LinkedRepos, repoID)
					return nil
				},
			})
			continue
		}
		if repo.WorkspaceID != workspaceID {
			out = append(out, Finding{
				Problem: fmt.Sprintf("linked_repos holds %s which belongs to another workspace", repoLabel(repo, repoID)),
				Repair:  "drop that entry from linked_repos",
				Apply: func() error {
					ws.LinkedRepos = filterOut(ws.LinkedRepos, repoID)
					return nil
				},
			})
		}
	}

	for repoID, repo := range s.Repositories {
		if repo == nil {
			continue
		}
		repoID, repo := repoID, repo
		if repo.WorkspaceID == workspaceID && !containsID(ws.LinkedRepos, repoID) {
			out = append(out, Finding{
				Problem: fmt.Sprintf("repository %s points to this workspace but is missing from linked_repos", repoLabel(repo, repoID)),
				Repair:  "add it to linked_repos",
				Apply: func() error {
					if !containsID(ws.LinkedRepos, repoID) {
						ws.LinkedRepos = append(ws.LinkedRepos, repoID)
					}
					return nil
				},
			})
		}
		if repo.WorkspaceID != "" {
			if _, err := shared.GetWorkspace(s, repo.WorkspaceID); err != nil {
				out = append(out, Finding{
					Problem: fmt.Sprintf("repository %s points to unknown workspace %s", repoLabel(repo, repoID), repo.WorkspaceID),
					Repair:  "clear workspace_id, or remove the entry",
					Decide:  true,
					Apply: func() error {
						return repairOrphanWorkspaceRef(s, repoID, p)
					},
				})
			}
		}
	}

	if namespacesNeedNormalize(ws.Namespaces) {
		out = append(out, Finding{
			Problem: "namespaces hold blank, untrimmed, or duplicated values",
			Repair:  "normalize and dedupe namespaces",
			Apply: func() error {
				ws.Namespaces = normalizeNamespaces(ws.Namespaces)
				return nil
			},
		})
	}
	return out
}

func repairOrphanWorkspaceRef(s *shared.State, repoID string, p prompt.Prompter) error {
	choice, err := p.Pick("Orphan workspace reference", []prompt.Choice{
		{Value: "clear", Description: "Clear workspace_id"},
		{Value: "remove", Description: "Remove the registry entry"},
	}, "clear")
	if err != nil {
		return err
	}
	switch choice {
	case "remove":
		return shared.DeleteRepo(s, repoID)
	case "clear":
		repo, err := shared.GetRepo(s, repoID)
		if err != nil {
			return ErrSkipped
		}
		old := repo.WorkspaceID
		repo.WorkspaceID = ""
		if ws, ok := s.Workspaces[old]; ok && ws != nil {
			ws.LinkedRepos = filterOut(ws.LinkedRepos, repoID)
		}
		return nil
	default:
		return fmt.Errorf("orphan workspace reference must be cleared or removed")
	}
}

func LinkedRepos(s *shared.State, workspaceID string, p prompt.Prompter) []Finding {
	ws, err := shared.GetWorkspace(s, workspaceID)
	if err != nil {
		return nil
	}
	var out []Finding
	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()

	for _, repoID := range append([]string(nil), ws.LinkedRepos...) {
		repo, err := shared.GetRepo(s, repoID)
		if err != nil {
			continue
		}
		repoID, repo := repoID, repo
		label := repoLabel(repo, repoID)

		if _, err := os.Stat(repo.CurrentPath); err != nil {
			out = append(out, Finding{
				Problem: fmt.Sprintf("%s: path missing (%s)", label, repo.CurrentPath),
				Repair:  "set a new path, remove the entry, or ignore",
				Decide:  true,
				Apply: func() error {
					return repairMissingPath(s, repoID, p, "Missing path")
				},
			})
			continue
		}
		if !isGitWorkTree(repo.CurrentPath) {
			out = append(out, Finding{
				Problem: fmt.Sprintf("%s: path is not a git work tree (%s)", label, repo.CurrentPath),
				Repair:  "remove the entry, or ignore",
				Decide:  true,
				Apply: func() error {
					return repairNotGitWorkTree(s, repoID, p)
				},
			})
			continue
		}
		if err := os.Chdir(repo.CurrentPath); err != nil {
			out = append(out, Finding{
				Problem: fmt.Sprintf("%s: path inaccessible (%s)", label, repo.CurrentPath),
				Repair:  "set a new path, remove the entry, or ignore",
				Decide:  true,
				Apply: func() error {
					return repairMissingPath(s, repoID, p, "Inaccessible path")
				},
			})
			continue
		}
		localID, _ := repoid.ReadLocal()
		if localID == "" {
			out = append(out, Finding{
				Problem: fmt.Sprintf("%s: elegant-git.repo-id is unset", label),
				Repair:  "stamp the registry id",
				Apply: func() error {
					return stampRepoIDAt(repo.CurrentPath, repoID)
				},
			})
		} else if localID != repoID {
			out = append(out, Finding{
				Problem: fmt.Sprintf("%s: elegant-git.repo-id %s differs from registry %s", label, localID, repoID),
				Repair:  "stamp the registry id",
				Apply: func() error {
					return stampRepoIDAt(repo.CurrentPath, repoID)
				},
			})
		}
		gitDir, err := memrepo.GitDir()
		if err != nil {
			continue
		}
		perRepo, err := memrepo.Load(gitDir)
		if err != nil {
			continue
		}
		if perRepo.WorkspaceID != "" && perRepo.WorkspaceID != workspaceID {
			out = append(out, Finding{
				Problem: fmt.Sprintf("%s: per-repo memory workspace_id differs from registry", label),
				Repair:  "rewrite per-repo memory workspace_id",
				Apply: func() error {
					return rewritePerRepoWorkspace(repo.CurrentPath, workspaceID, repoID)
				},
			})
		}
	}
	return out
}

func repairMissingPath(s *shared.State, repoID string, p prompt.Prompter, label string) error {
	if _, err := shared.GetRepo(s, repoID); err != nil {
		return ErrSkipped
	}
	choice, err := p.Pick(label, []prompt.Choice{
		{Value: "path", Description: "Provide a new git work tree path"},
		{Value: "remove", Description: "Remove the registry entry"},
		{Value: "ignore", Description: "Leave as-is"},
	}, "path")
	if err != nil {
		return skipOrErr(err)
	}
	switch choice {
	case "ignore":
		return ErrSkipped
	case "remove":
		return shared.DeleteRepo(s, repoID)
	case "path":
		return relocateMissingPath(s, repoID, p)
	default:
		return ErrSkipped
	}
}

func repairNotGitWorkTree(s *shared.State, repoID string, p prompt.Prompter) error {
	if _, err := shared.GetRepo(s, repoID); err != nil {
		return ErrSkipped
	}
	choice, err := p.Pick("Not a git work tree", []prompt.Choice{
		{Value: "remove", Description: "Remove the registry entry"},
		{Value: "ignore", Description: "Leave as-is"},
	}, "remove")
	if err != nil {
		return skipOrErr(err)
	}
	if choice == "remove" {
		return shared.DeleteRepo(s, repoID)
	}
	return ErrSkipped
}

func relocateMissingPath(s *shared.State, repoID string, p prompt.Prompter) error {
	for {
		path, err := p.String("New path", "")
		if err != nil {
			return skipOrErr(err)
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			text.ErrorText("invalid path")
			continue
		}
		if _, err := os.Stat(abs); err != nil {
			text.ErrorText("path does not exist")
			continue
		}
		if !isGitWorkTree(abs) {
			text.ErrorText("not a git work tree")
			continue
		}
		if owner := pathOwnedByOther(s, repoID, abs); owner != "" {
			text.ErrorText("already registered as " + owner)
			continue
		}
		localID, err := localRepoIDAt(abs)
		if err != nil {
			text.ErrorText("cannot read " + repoid.Key)
			continue
		}
		if localID != "" && localID != repoID {
			text.ErrorText("path already has " + repoid.Key + " " + localID)
			continue
		}
		if err := shared.RecordPath(s, repoID, abs); err != nil {
			return err
		}
		if localID == repoID {
			return nil
		}
		return stampRepoIDAt(abs, repoID)
	}
}

func skipOrErr(err error) error {
	if errors.Is(err, prompt.ErrUserCancelled) || errors.Is(err, io.EOF) {
		return ErrSkipped
	}
	return err
}

func pathOwnedByOther(s *shared.State, repoID, absPath string) string {
	for id, repo := range s.Repositories {
		if repo == nil || id == repoID {
			continue
		}
		if repo.CurrentPath == absPath {
			return repoLabel(repo, id)
		}
	}
	return ""
}

func localRepoIDAt(absPath string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(absPath); err != nil {
		return "", err
	}
	return repoid.ReadLocal()
}

func stampRepoIDAt(absPath, registryID string) error {
	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(absPath); err != nil {
		return err
	}
	return repoid.StampLocal(registryID)
}

func rewritePerRepoWorkspace(absPath, workspaceID, repoID string) error {
	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(absPath); err != nil {
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
	perRepo.WorkspaceID = workspaceID
	if perRepo.RepoID == "" {
		perRepo.RepoID = repoID
	}
	return memrepo.Save(gitDir, perRepo)
}

func isGitWorkTree(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir() || info.Mode().IsRegular()
}

func repoLabel(repo *shared.Repository, id string) string {
	if repo != nil && repo.Name != "" {
		return repo.Name
	}
	return id
}

func filterOut(ids []string, drop string) []string {
	var out []string
	for _, id := range ids {
		if id != drop {
			out = append(out, id)
		}
	}
	return out
}

func containsID(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func namespacesNeedNormalize(ns []string) bool {
	seen := map[string]bool{}
	for _, n := range ns {
		trimmed := strings.TrimSpace(n)
		if trimmed == "" || trimmed != n || seen[trimmed] {
			return true
		}
		seen[trimmed] = true
	}
	return false
}

func normalizeNamespaces(ns []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, n := range ns {
		n = strings.TrimSpace(n)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}
