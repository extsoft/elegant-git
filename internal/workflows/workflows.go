// Package workflows runs optional ahead/after hook scripts for a command.
package workflows

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/text"
)

// Skip when true disables ahead/after hook execution (--no-workflows).
var Skip bool

// RunAhead executes personal then common <command>-ahead workflow files.
func RunAhead(command string) {
	runHook(command, "ahead")
}

// RunAfter executes personal then common <command>-after workflow files.
func RunAfter(command string) {
	runHook(command, "after")
}

func runHook(command, hookType string) {
	if Skip {
		return
	}
	runFile(PersonalWorkflowsFile(command, hookType))
	runFile(CommonWorkflowsFile(command, hookType))
}

func runFile(path string) {
	if path == "" {
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return
	}
	text.CommandText(path)
	cmd := exec.Command("bash", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Dir = repoRootFunc()
	_ = cmd.Run()
}

var repoRootFunc = defaultRepoRoot

func defaultRepoRoot() string {
	out := git.OutputOK("rev-parse", "--show-toplevel")
	if out == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "."
		}
		return wd
	}
	return strings.TrimSpace(out)
}

// Prefix returns the repository-relative prefix for workflow paths.
func Prefix(command string) string {
	if command == "init-repository" || command == "clone-repository" {
		return ""
	}
	out := git.OutputOK("rev-parse", "--show-cdup")
	return strings.TrimSpace(out)
}

// WorkflowsDirectory returns the directory for personal or common workflows.
func WorkflowsDirectory(location, command string) (string, error) {
	switch location {
	case "personal":
		return filepath.Join(Prefix(command), ".git", ".workflows"), nil
	case "common":
		return filepath.Join(Prefix(command), ".workflows"), nil
	default:
		return "", fmt.Errorf("unknown workflows location: %s", location)
	}
}

// WorkflowsFile returns the path to a workflow hook file.
func WorkflowsFile(location, command, hookType string) (string, error) {
	dir, err := WorkflowsDirectory(location, command)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, command+"-"+hookType), nil
}

// PersonalWorkflowsFile returns .git/.workflows/<command>-<type>.
func PersonalWorkflowsFile(command, hookType string) string {
	p, err := WorkflowsFile("personal", command, hookType)
	if err != nil {
		return ""
	}
	return p
}

// CommonWorkflowsFile returns .workflows/<command>-<type>.
func CommonWorkflowsFile(command, hookType string) string {
	p, err := WorkflowsFile("common", command, hookType)
	if err != nil {
		return ""
	}
	return p
}
