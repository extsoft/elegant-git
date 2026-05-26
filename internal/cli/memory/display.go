package memory

import (
	"fmt"
	"io"
	"os"

	"github.com/bees-hive/elegant-git/internal/git"
	memrepo "github.com/bees-hive/elegant-git/internal/memory/repo"
	"github.com/bees-hive/elegant-git/internal/memory/repoid"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/version"
)

// PrintMemorySummary prints paths, counts, and pointers to status commands.
func PrintMemorySummary(w io.Writer) error {
	fmt.Fprintf(w, "version: %s\n", version.Version)

	sharedPath, err := shared.Path()
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "shared memory: %s\n", fileStatusLine(sharedPath))
	if v := os.Getenv("ELEGANT_GIT_STATE_FILE"); v != "" {
		fmt.Fprintf(w, "  (ELEGANT_GIT_STATE_FILE=%s)\n", v)
	}

	s, err := shared.Load()
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "  profiles: %d\n", len(s.Profiles))
	fmt.Fprintf(w, "  repositories: %d\n", reposWithProfile(s))

	_, gitErr := memrepo.GitDir()
	if gitErr != nil {
		fmt.Fprintln(w, "repository: (not inside a git work tree)")
	} else {
		repoID, _ := repoid.ReadLocal()
		if repoID == "" {
			fmt.Fprintln(w, "repository: inside git work tree (not configured; run repo configure)")
		} else if reg, err := shared.GetRepo(s, repoID); err == nil {
			fmt.Fprintf(w, "repository: %s (%s)\n", reg.Name, reg.CurrentPath)
		} else {
			fmt.Fprintf(w, "repository: repo-id %s (not in registry)\n", repoID)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "For details, run:")
	fmt.Fprintln(w, "  git elegant git status")
	fmt.Fprintln(w, "  git elegant repo status")
	fmt.Fprintln(w, "  git elegant profile status")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Catalogs:")
	fmt.Fprintln(w, "  git elegant memory profiles")
	fmt.Fprintln(w, "  git elegant memory repositories")
	return nil
}

// PrintGitStatus prints global install and shared memory state (no profile dump).
func PrintGitStatus(w io.Writer) error {
	return printGitState(w)
}

// PrintRepoStatus prints per-repo memory and registry for the current repository.
func PrintRepoStatus(w io.Writer) error {
	return printRepoState(w)
}

// PrintProfileStatus prints the linked profile for the current repository.
func PrintProfileStatus(w io.Writer) error {
	gitDir, gitErr := memrepo.GitDir()
	if gitErr != nil {
		fmt.Fprintln(w, "repository: (not inside a git work tree)")
		return nil
	}

	s, err := shared.Load()
	if err != nil {
		return err
	}

	repoID, _ := repoid.ReadLocal()
	if repoID == "" {
		fmt.Fprintln(w, "linked profile: (not set; run repo configure)")
		return nil
	}

	reg, err := shared.GetRepo(s, repoID)
	if err != nil {
		fmt.Fprintf(w, "linked profile: repo-id %s not in registry\n", repoID)
		return nil
	}

	prof, err := shared.GetProfile(s, reg.ProfileID)
	if err != nil {
		fmt.Fprintf(w, "linked profile: id %s (missing from shared memory)\n", reg.ProfileID)
		return nil
	}

	printProfileFields(w, "", reg.ProfileID, prof)
	printLinkedRepos(w, s, prof, "")

	perRepo, err := memrepo.Load(gitDir)
	if err != nil {
		return err
	}
	if perRepo.ProfileID != "" && perRepo.ProfileID != reg.ProfileID {
		if alt, err := shared.GetProfile(s, perRepo.ProfileID); err == nil {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "per-repo memory profile:")
			printProfileFields(w, "  ", perRepo.ProfileID, alt)
		} else {
			fmt.Fprintf(w, "\nper-repo memory profile_id: %s\n", perRepo.ProfileID)
		}
	}
	return nil
}

func printGitState(w io.Writer) error {
	fmt.Fprintf(w, "version: %s\n", version.Version)

	sharedPath, err := shared.Path()
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "shared memory: %s\n", fileStatusLine(sharedPath))
	if v := os.Getenv("ELEGANT_GIT_STATE_FILE"); v != "" {
		fmt.Fprintf(w, "  (ELEGANT_GIT_STATE_FILE=%s)\n", v)
	}

	s, err := shared.Load()
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "  profiles: %d\n", len(s.Profiles))
	fmt.Fprintf(w, "  repositories: %d\n", reposWithProfile(s))

	fmt.Fprintln(w, "global git identity:")
	printGitIdentity(w, "  ", git.ConfigGlobalGet)

	if acquired := git.ConfigGlobalGet("elegant-git.acquired"); acquired != "" {
		fmt.Fprintf(w, "  elegant-git.acquired: %s\n", acquired)
	} else {
		fmt.Fprintln(w, "  elegant-git.acquired: (not set; run git configure)")
	}

	return nil
}

func printRepoState(w io.Writer) error {
	gitDir, gitErr := memrepo.GitDir()
	if gitErr != nil {
		fmt.Fprintln(w, "repository: (not inside a git work tree)")
		return nil
	}

	s, err := shared.Load()
	if err != nil {
		return err
	}

	repoPath := memrepo.Path(gitDir)
	fmt.Fprintf(w, "per-repo memory: %s\n", fileStatusLine(repoPath))
	if v := os.Getenv("ELEGANT_GIT_REPO_STATE_FILE"); v != "" {
		fmt.Fprintf(w, "  (ELEGANT_GIT_REPO_STATE_FILE=%s)\n", v)
	}

	repoID, _ := repoid.ReadLocal()
	if repoID != "" {
		fmt.Fprintf(w, "  elegant-git.repo-id: %s\n", repoID)
		if reg, err := shared.GetRepo(s, repoID); err == nil {
			fmt.Fprintf(w, "  registry name: %s\n", reg.Name)
			fmt.Fprintf(w, "  registry path: %s\n", reg.CurrentPath)
			if reg.OriginURL != "" {
				fmt.Fprintf(w, "  origin: %s\n", reg.OriginURL)
			}
			if prof, err := shared.GetProfile(s, reg.ProfileID); err == nil {
				fmt.Fprintf(w, "  linked profile: %s\n", prof.Name)
			} else {
				fmt.Fprintf(w, "  profile id: %s (missing from shared memory)\n", reg.ProfileID)
			}
		}
	} else {
		fmt.Fprintln(w, "  elegant-git.repo-id: (not set; run repo configure)")
	}

	perRepo, err := memrepo.Load(gitDir)
	registryProfileID := ""
	if repoID != "" {
		if reg, err := shared.GetRepo(s, repoID); err == nil {
			registryProfileID = reg.ProfileID
		}
	}
	if err == nil {
		if perRepo.RepoID != "" && perRepo.RepoID != repoID {
			fmt.Fprintf(w, "  memory repo_id: %s\n", perRepo.RepoID)
		}
		if perRepo.ProfileID != "" && perRepo.ProfileID != registryProfileID {
			if prof, err := shared.GetProfile(s, perRepo.ProfileID); err == nil {
				fmt.Fprintf(w, "  per-repo memory profile: %s\n", prof.Name)
			} else {
				fmt.Fprintf(w, "  per-repo memory profile_id: %s\n", perRepo.ProfileID)
			}
		}
		if perRepo.DefaultBranch != "" {
			fmt.Fprintf(w, "  default branch: %s\n", perRepo.DefaultBranch)
		}
		if len(perRepo.ProtectedBranches) > 0 {
			fmt.Fprintf(w, "  protected branches: %v\n", perRepo.ProtectedBranches)
		}
	}

	fmt.Fprintln(w, "local git identity:")
	printGitIdentity(w, "  ", git.ConfigLocalGet)
	return nil
}

func printProfileFields(w io.Writer, prefix, id string, p *shared.Profile) {
	fmt.Fprintf(w, "%sname:         %s\n", prefix, p.Name)
	fmt.Fprintf(w, "%sid:           %s\n", prefix, id)
	fmt.Fprintf(w, "%suser.name:    %s\n", prefix, p.UserName)
	fmt.Fprintf(w, "%suser.email:   %s\n", prefix, p.UserEmail)
	fmt.Fprintf(w, "%ssigning key:  %s\n", prefix, showOptional(p.SigningKey))
	fmt.Fprintf(w, "%sgpg program:  %s\n", prefix, showOptional(p.GPGProgram))
	fmt.Fprintf(w, "%seditor:       %s\n", prefix, showOptional(p.Editor))
}

func printLinkedRepos(w io.Writer, s *shared.State, p *shared.Profile, prefix string) {
	if len(p.LinkedRepos) == 0 {
		fmt.Fprintf(w, "%slinked repos: (none)\n", prefix)
		return
	}
	fmt.Fprintf(w, "%slinked repos: %d\n", prefix, len(p.LinkedRepos))
	for _, repoID := range p.LinkedRepos {
		repo, err := shared.GetRepo(s, repoID)
		if err != nil {
			fmt.Fprintf(w, "%s  - %s\n", prefix, repoID)
			continue
		}
		fmt.Fprintf(w, "%s  - %s    %s\n", prefix, repo.Name, repo.CurrentPath)
	}
}

func printGitIdentity(w io.Writer, prefix string, get func(string) string) {
	fmt.Fprintf(w, "%suser.name:    %s\n", prefix, showOptional(get("user.name")))
	fmt.Fprintf(w, "%suser.email:   %s\n", prefix, showOptional(get("user.email")))
	fmt.Fprintf(w, "%ssigning key:  %s\n", prefix, showOptional(get("user.signingkey")))
	fmt.Fprintf(w, "%sgpg program:  %s\n", prefix, showOptional(get("gpg.program")))
	fmt.Fprintf(w, "%seditor:       %s\n", prefix, showOptional(get("core.editor")))
}

func showOptional(v string) string {
	if v == "" {
		return "(unset)"
	}
	return v
}

func fileStatusLine(path string) string {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return path + " (not created yet)"
		}
		return path + " (unreadable)"
	}
	return path
}

func reposWithProfile(s *shared.State) int {
	n := 0
	for _, r := range s.Repositories {
		if r != nil && r.ProfileID != "" {
			n++
		}
	}
	return n
}
