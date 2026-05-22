package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/bees-hive/elegant-git/internal/exitcode"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/bees-hive/elegant-git/internal/workflows"
)

func branchFromRemoteBranch(remoteBranch string) string {
	if i := strings.Index(remoteBranch, "/"); i >= 0 {
		return remoteBranch[i+1:]
	}
	return remoteBranch
}

func remoteFromRemoteBranch(remoteBranch string) string {
	if i := strings.Index(remoteBranch, "/"); i >= 0 {
		return remoteBranch[:i]
	}
	return ""
}

func exitProtectedDeliver(branch string) {
	text.ErrorBox(fmt.Sprintf("The push of the protected '%s' branch is prohibited.", branch))
	text.ErrorText("Consider using 'git elegant accept-work' or use plain 'git push'.")
	os.Exit(exitcode.ProtectedBranch)
}

func openURLsIfPossible(output string) {
	if _, err := exec.LookPath("open"); err != nil {
		return
	}
	re := regexp.MustCompile(`https?://\S+`)
	for _, line := range strings.Fields(output) {
		if re.MatchString(line) || strings.HasPrefix(line, "http") {
			url := strings.TrimRight(line, ".,;)")
			if strings.HasPrefix(url, "http") {
				_ = exec.Command("open", url).Run()
			}
		}
	}
	for _, m := range re.FindAllString(output, -1) {
		_ = exec.Command("open", m).Run()
	}
}

func copyNotesIfPossible(notes string) {
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

func localBranchExists(name string) bool {
	_, err := git.Output("rev-parse", "--verify", "--quiet", "--abbrev-ref", "--branches=refs/heads", name)
	return err == nil
}

func branchUpstreamShort(branch string) string {
	return git.OutputOK("branch", "--list", "--format", "%(upstream:short)", branch)
}

func runObtainWorkHooks(fn func() error) error {
	workflows.RunAhead("obtain-work")
	defer workflows.RunAfter("obtain-work")
	return fn()
}

func readLineAnswer(prompt string) (string, error) {
	text.QuestionText(prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
