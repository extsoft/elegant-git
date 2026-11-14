package work

import (
	"errors"
	"fmt"
	"io"
	"strings"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/exitcode"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
)

func guardNewBranchName(p prompt.Prompter, name string) (string, error) {
	branches, err := state.LocalBranchNames()
	if err != nil {
		return "", err
	}
	return resolveNewBranchName(p, name, branches, "")
}

func resolveNewBranchName(p prompt.Prompter, name string, branches []string, reask string) (string, error) {
	return resolveNewBranchNameFold(p, name, branches, reask, nil)
}

func resolveNewBranchNameFold(p prompt.Prompter, name string, branches []string, reask string, fold *caseFold) (string, error) {
	name = strings.TrimSpace(name)
	if problem := submitNewBranchNameProblem(name, branches); problem != "" {
		text.ErrorText(problem)
		if prompt.NonInteractive(p) {
			return "", cliruntime.ExitStatus{Code: exitcode.WorkflowError}
		}
		ready := caseFold{}
		if fold != nil {
			ready = *fold
		} else {
			ready = prepareCaseFold()
		}
		return askNewBranchName(p, "Branch name", branches, reask, ready)
	}
	return name, nil
}

func askNewBranchName(p prompt.Prompter, label string, branches []string, initial string, fold caseFold) (string, error) {
	for {
		name, err := prompt.ValidatedLine(p, label, initial, func(s string) string {
			return liveBranchProblem(strings.TrimSpace(s), branches, fold)
		})
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", errors.New("end of input")
			}
			return "", err
		}
		name = strings.TrimSpace(name)
		initial = name
		if problem := submitNewBranchNameProblem(name, branches); problem != "" {
			text.ErrorText(problem)
			continue
		}
		return name, nil
	}
}

func liveBranchProblem(name string, branches []string, fold caseFold) string {
	if problem := git.NewBranchNameProblem(name, branches); problem != "" {
		return problem
	}
	if !fold.on {
		return ""
	}
	for _, existing := range branches {
		if existing == "" {
			continue
		}
		if strings.EqualFold(existing, name) {
			if existing != name && caseOnlyResolves(fold, name) {
				return fmt.Sprintf("branch '%s' already exists", name)
			}
			continue
		}
		if foldSlashPrefix(name, existing) || foldSlashPrefix(existing, name) {
			return fmt.Sprintf("branch '%s' conflicts with existing branch '%s'", name, existing)
		}
	}
	return ""
}

type caseFold struct {
	on   bool
	memo map[string]bool
}

func prepareCaseFold() caseFold {
	if !coreIgnoreCase() {
		return caseFold{}
	}
	return caseFold{on: true, memo: map[string]bool{}}
}

func caseOnlyResolves(fold caseFold, name string) bool {
	key := strings.ToLower(name)
	if resolved, ok := fold.memo[key]; ok {
		return resolved
	}
	_, err := git.Output("show-ref", "--verify", "--quiet", "refs/heads/"+name)
	resolved := err == nil
	fold.memo[key] = resolved
	return resolved
}

func submitNewBranchNameProblem(name string, branches []string) string {
	if problem := git.NewBranchNameProblem(name, branches); problem != "" {
		return problem
	}
	if problem := ignoreCaseProblem(name, branches); problem != "" {
		return problem
	}
	if err := git.CheckBranchName(name); err != nil {
		return err.Error()
	}
	return ""
}

func ignoreCaseProblem(name string, branches []string) string {
	if !coreIgnoreCase() {
		return ""
	}
	for _, existing := range branches {
		if existing == "" || strings.EqualFold(existing, name) {
			continue
		}
		if foldSlashPrefix(name, existing) || foldSlashPrefix(existing, name) {
			return fmt.Sprintf("branch '%s' conflicts with existing branch '%s'", name, existing)
		}
	}
	if _, err := git.Output("show-ref", "--verify", "--quiet", "refs/heads/"+name); err == nil {
		return fmt.Sprintf("branch '%s' already exists", name)
	}
	return ""
}

func foldSlashPrefix(name, prefix string) bool {
	return strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)+"/")
}

func coreIgnoreCase() bool {
	out, err := git.Output("config", "--get", "core.ignorecase")
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(out)) {
	case "true", "yes", "on", "1":
		return true
	default:
		return false
	}
}
