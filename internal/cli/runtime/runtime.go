// Package runtime provides shared CLI execution helpers.
package runtime

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/exitcode"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/extsoft/elegant-git/internal/workflows"
	"github.com/spf13/cobra"
)

const SiteURL = "https://elegant-git.extsoft.pro"

type stdinKeyType struct{}

var stdinKey = stdinKeyType{}

// WithStdin stores reader on ctx for interactive commands.
func WithStdin(ctx context.Context, r io.Reader) context.Context {
	return context.WithValue(ctx, stdinKey, r)
}

// StdinFromContext returns stdin from ctx or os.Stdin.
func StdinFromContext(ctx context.Context) io.Reader {
	if r, ok := ctx.Value(stdinKey).(io.Reader); ok && r != nil {
		return r
	}
	return os.Stdin
}

// RunWithWorkflows runs ahead/after hooks for the command's canonical id.
func RunWithWorkflows(cmd *cobra.Command, id cmdid.ID, fn func() error) error {
	ctx := cmd.Context()
	workflows.RunAhead(ctx, id)
	defer workflows.RunAfter(ctx, id)
	return fn()
}

// RunWithCompat runs hooks for id and additionally for a legacy hook name (DEP-008).
func RunWithCompat(cmd *cobra.Command, id cmdid.ID, legacyAlso string, fn func() error) error {
	ctx := cmd.Context()
	workflows.RunAheadCompat(ctx, id, legacyAlso)
	defer workflows.RunAfterCompat(ctx, id, legacyAlso)
	return fn()
}

func ExitWorkflowError(messages ...string) {
	for _, m := range messages {
		text.ErrorText(m)
	}
	os.Exit(exitcode.WorkflowError)
}

func ShellVerbose(name string, args ...string) error {
	text.CommandText(append([]string{name}, args...)...)
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func ReadLineAnswer(ctx context.Context, prompt string) (string, error) {
	text.QuestionText(prompt)
	line, err := bufio.NewReader(StdinFromContext(ctx)).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// StdinIsInteractive reports whether stdin is a character device (TTY).
func StdinIsInteractive(ctx context.Context) bool {
	f, ok := StdinFromContext(ctx).(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func CurrentBranch() string {
	return git.OutputOK("rev-parse", "--abbrev-ref", "HEAD")
}

func ExitProtectedNoCommits(branch string) {
	text.ErrorBox(fmt.Sprintf("No direct commits to the protected '%s' branch.", branch))
	text.ErrorText("Please read more on " + SiteURL + ".")
	text.ErrorText("Run 'git elegant work start' prior to retrying this command.")
	os.Exit(exitcode.ProtectedBranch)
}

func ExitProtectedNoRewrite(branch string) {
	text.ErrorBox(fmt.Sprintf("The protected '%s' branch history can't be rewritten.", branch))
	text.ErrorText("Please read more on " + SiteURL + ".")
	os.Exit(exitcode.ProtectedBranch)
}

func ExitProtectedDeliver(branch string) {
	text.ErrorBox(fmt.Sprintf("The push of the protected '%s' branch is prohibited.", branch))
	text.ErrorText("Consider using 'git elegant work accept' or use plain 'git push'.")
	os.Exit(exitcode.ProtectedBranch)
}

func PullOrInform() {
	if err := git.Verbose("pull"); err != nil {
		text.InfoText("As the pull can't be completed, the current local version is used.")
	}
}

func FetchOrInform() {
	if err := git.Verbose("fetch"); err != nil {
		text.InfoText("Unable to fetch. The last local revision will be used.")
	}
}

func BranchFromRemoteBranch(remoteBranch string) string {
	if i := strings.Index(remoteBranch, "/"); i >= 0 {
		return remoteBranch[i+1:]
	}
	return remoteBranch
}

func RemoteFromRemoteBranch(remoteBranch string) string {
	if i := strings.Index(remoteBranch, "/"); i >= 0 {
		return remoteBranch[:i]
	}
	return ""
}

func LocalBranchExists(name string) bool {
	_, err := git.Output("rev-parse", "--verify", "--quiet", "--abbrev-ref", "--branches=refs/heads", name)
	return err == nil
}

func BranchUpstreamShort(branch string) string {
	return state.UpstreamOf(branch)
}

func OpenURLsIfPossible(output string) {
	if _, err := exec.LookPath("open"); err != nil {
		return
	}
	for _, line := range strings.Fields(output) {
		url := strings.TrimRight(line, ".,;)")
		if strings.HasPrefix(url, "http") {
			_ = exec.Command("open", url).Run()
		}
	}
}

func CopyNotesIfPossible(notes string) {
	tool := []string{"cat"}
	if _, err := exec.LookPath("pbcopy"); err == nil {
		tool = []string{"pbcopy"}
	} else if _, err := exec.LookPath("xclip"); err == nil {
		tool = []string{"xclip", "-selection", "clipboard"}
	}
	cmd := exec.Command(tool[0], tool[1:]...)
	cmd.Stdin = strings.NewReader(notes)
	cmd.Stdout = os.Stdout
	_ = cmd.Run()
	if tool[0] != "cat" {
		text.InfoText("The release notes are copied to clipboard.")
	} else {
		fmt.Fprint(os.Stdout, notes)
	}
}

func GitStdout(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ArgAt(args []string, i int) string {
	if len(args) > i {
		return args[i]
	}
	return ""
}
