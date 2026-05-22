package cli

import (
	"fmt"
	"os"

	"github.com/bees-hive/elegant-git/internal/exitcode"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/text"
)

func currentBranch() string {
	return git.OutputOK("rev-parse", "--abbrev-ref", "HEAD")
}

func exitProtectedNoCommits(branch string) {
	text.ErrorBox(fmt.Sprintf("No direct commits to the protected '%s' branch.", branch))
	text.ErrorText("Please read more on " + siteURL + ".")
	text.ErrorText("Run 'git elegant start-work' prior to retrying this command.")
	os.Exit(exitcode.ProtectedBranch)
}

func exitProtectedNoRewrite(branch string) {
	text.ErrorBox(fmt.Sprintf("The protected '%s' branch history can't be rewritten.", branch))
	text.ErrorText("Please read more on " + siteURL + ".")
	os.Exit(exitcode.ProtectedBranch)
}

func pullOrInform() {
	if err := git.Verbose("pull"); err != nil {
		text.InfoText("As the pull can't be completed, the current local version is used.")
	}
}

func fetchOrInform() {
	if err := git.Verbose("fetch"); err != nil {
		text.InfoText("Unable to fetch. The last local revision will be used.")
	}
}
