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
	return resolveNewBranchName(p, name, branches)
}

func resolveNewBranchName(p prompt.Prompter, name string, branches []string) (string, error) {
	name = strings.TrimSpace(name)
	if problem := submitNewBranchNameProblem(name, branches); problem != "" {
		text.ErrorText(problem)
		if prompt.NonInteractive(p) {
			return "", cliruntime.ExitStatus{Code: exitcode.WorkflowError}
		}
		return askNewBranchName(p, "Branch name", branches)
	}
	return name, nil
}

func askNewBranchName(p prompt.Prompter, label string, branches []string) (string, error) {
	for {
		name, err := readBranchName(p, label)
		if err != nil {
			return "", err
		}
		name = strings.TrimSpace(name)
		if problem := submitNewBranchNameProblem(name, branches); problem != "" {
			text.ErrorText(problem)
			continue
		}
		return name, nil
	}
}

func readBranchName(p prompt.Prompter, label string) (string, error) {
	name, err := p.String(label, "")
	if errors.Is(err, io.EOF) {
		return "", errors.New("end of input")
	}
	return name, err
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
