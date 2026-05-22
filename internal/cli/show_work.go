package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/state"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newShowWorkCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWithWorkflows(cmd, showWorkRun)
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func showWorkRun() error {
	branch := currentBranch()
	text.InfoText(">>> Branch refs:")
	text.InfoText("local:  " + branch)
	if upstream := state.UpstreamOf(branch); upstream != "" {
		text.InfoText("remote: " + upstream)
	}
	text.InfoText("")

	latest := config.FreshestDefaultBranch()
	if git.OutputOK("rev-list", latest+".."+branch) != "" {
		text.InfoText(fmt.Sprintf(">>> New commits (comparing to '%s' branch):", latest))
		if err := gitStdout("log", "--oneline", latest+".."+branch); err != nil {
			return err
		}
		text.InfoText("")
	}
	if status := git.OutputOK("status", "--short"); status != "" {
		text.InfoText(">>> Uncommitted modifications:")
		if err := gitStdout("status", "--short"); err != nil {
			return err
		}
		text.InfoText("")
	}
	if stashes := git.OutputOK("stash", "list"); stashes != "" {
		text.InfoText(">>> Available stashes:")
		if err := gitStdout("stash", "list"); err != nil {
			return err
		}
		text.InfoText("")
	}
	return nil
}

func gitStdout(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
