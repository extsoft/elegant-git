package repo

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/exitcode"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

const initialCommitMessage = `Add initial empty commit

This commit is the first commit in this working tree. It does not have
any changes. However, it simplifies further work at least in the
following cases:
- it's possible to create a branch now
- it's possible to manage the second commit if it requires some
polishing after creation

This commit is created automatically by Elegant Git after the
initialization of a new repository.
`

const repositoryProbe = "rev-parse --is-inside-work-tree --is-bare-repository --is-inside-git-dir"

const freshRepositoryText = "fatal: not a git repository (or any of the parent directories):"

const freshMountText = "fatal: not a git repository (or any parent up to mount point "

var initID = cmdid.ID{Command: "repo", Action: "init"}

func newInitCommand() *cobra.Command {
	var workspaceName string
	spec := configureSpec(&workspaceName)
	c := &cobra.Command{
		Use:   "init <workspace>",
		Short: "Initializes a new repository and configures it",
		Long: "Creates a git repository in the current directory, configures it, and records an empty initial commit. " +
			"Refuses when this directory is already a git repository or inside one, including an unborn, linked, or bare repository, a .git file or directory, and a subdirectory of a worktree. " +
			"Leaves ordinary files untracked and does not stage them. " +
			"The refusal happens before hooks or workspace prompts. " +
			"If configuration or the commit stops after initialization, the new repository and its files are left in place.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := refuseReinitialization(); err != nil {
				return err
			}
			return cliruntime.RunWithCompat(cmd, initID, "acquire-repository", func() error {
				if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
					return err
				}
				return initRun(cmd, workspaceName)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func initRun(cmd *cobra.Command, workspace string) error {
	if err := refuseReinitialization(); err != nil {
		return err
	}
	if err := git.Verbose("init"); err != nil {
		return err
	}
	if err := ConfigureRun(cmd, workspace); err != nil {
		reportPartialInit(
			"Initialized a git repository.",
			"Configuration did not finish, so the initial commit was not created.",
		)
		return err
	}
	if err := commitEmptyInitial(); err != nil {
		reportPartialInit(
			"Initialized a git repository and configured it.",
			"The initial commit was not created.",
		)
		return err
	}
	if err := git.Verbose("show"); err != nil {
		reportPartialInit(
			"Initialized a git repository, configured it, and recorded the initial commit.",
			"The commit could not be shown.",
		)
		return err
	}
	return nil
}

func reportPartialInit(done, skipped string) {
	text.InfoText(done)
	text.ErrorText(skipped)
	text.ErrorText("The new repository and its files were left in place.")
}

func refuseReinitialization() error {
	if _, err := os.Lstat(".git"); err == nil {
		return existingRepositoryRefusal()
	} else if err != nil && !os.IsNotExist(err) {
		return reportDetectionFailure("", err)
	}
	out, err := git.OutputC(strings.Fields(repositoryProbe)...)
	if err != nil {
		if isFreshDirectory(out, err) {
			return nil
		}
		return reportDetectionFailure(out, err)
	}
	return existingRepositoryRefusal()
}

func existingRepositoryRefusal() error {
	text.ErrorText("This directory is already a git repository or is inside one, so it was not initialized.")
	return cliruntime.ExitStatus{Code: exitcode.WorkflowError}
}

func isFreshDirectory(out string, err error) bool {
	msg := out
	if err != nil {
		msg += "\n" + err.Error()
	}
	return strings.Contains(msg, freshRepositoryText) || strings.Contains(msg, freshMountText)
}

func reportDetectionFailure(out string, err error) error {
	if msg := strings.TrimSpace(out); msg != "" {
		text.ErrorText(msg)
	} else if err != nil && strings.TrimSpace(err.Error()) != "" {
		text.ErrorText(err.Error())
	}
	text.ErrorText("Could not tell whether this directory is a git repository, so it was not initialized.")
	return gitExit(err)
}

func commitEmptyInitial() error {
	return writeEmptyInitialCommit(nil)
}

func writeEmptyInitialCommit(afterWrite func() error) error {
	if err := refuseStagedIndex(); err != nil {
		return err
	}
	msg, err := writeInitialCommitMessage()
	if err != nil {
		return err
	}
	defer os.Remove(msg)
	tree, err := git.OutputC("mktree")
	if err != nil {
		return gitFailure(tree, err)
	}
	treeID, err := oneObjectID(tree)
	if err != nil {
		return err
	}
	commit, err := git.OutputC("commit-tree", treeID, "-F", msg)
	if err != nil {
		return gitFailure(commit, err)
	}
	commitID, err := oneObjectID(commit)
	if err != nil {
		return err
	}
	if afterWrite != nil {
		if err := afterWrite(); err != nil {
			return err
		}
	}
	if err := refuseStagedIndex(); err != nil {
		return err
	}
	return git.Verbose("update-ref", "HEAD", commitID)
}

func refuseStagedIndex() error {
	entries, err := git.OutputC("ls-files", "--stage")
	if err != nil {
		fail := gitFailure(entries, err)
		text.ErrorText("Could not read the index, so the initial commit was not created.")
		return fail
	}
	if strings.TrimSpace(entries) == "" {
		return nil
	}
	text.ErrorText("The index already has staged entries, so the initial commit was not created.")
	text.ErrorText("Those entries were left as they are.")
	return cliruntime.ExitStatus{Code: exitcode.WorkflowError}
}

func writeInitialCommitMessage() (string, error) {
	dir := os.TempDir()
	if err := messageOutsideWorktree(dir); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, "eg-init-message-*")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if _, err := f.WriteString(initialCommitMessage); err != nil {
		_ = f.Close()
		_ = os.Remove(name)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	if err := messageOutsideWorktree(name); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}

func messageOutsideWorktree(path string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	inside, err := pathInside(cwd, path)
	if err != nil {
		return err
	}
	if inside {
		return fmt.Errorf("commit message path %s is inside the worktree", path)
	}
	return nil
}

func pathInside(dir, path string) (bool, error) {
	absDir, err := resolvePath(dir)
	if err != nil {
		return false, err
	}
	absPath, err := resolvePath(path)
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return false, err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false, nil
	}
	return true, nil
}

func resolvePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

func oneObjectID(out string) (string, error) {
	id := strings.TrimSpace(out)
	if !objectID(id) {
		return "", fmt.Errorf("git reported %q instead of an object id", id)
	}
	return id, nil
}

func objectID(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func gitFailure(out string, err error) error {
	if msg := strings.TrimSpace(out); msg != "" {
		text.ErrorText(msg)
	}
	return gitExit(err)
}

func gitExit(err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return cliruntime.ExitStatus{Code: exitErr.ExitCode()}
	}
	if err != nil {
		return err
	}
	return cliruntime.ExitStatus{Code: 1}
}
