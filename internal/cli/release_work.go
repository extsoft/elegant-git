package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/pipe"
	"github.com/bees-hive/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

func newReleaseWorkCommand(spec commandSpec) *cobra.Command {
	c := &cobra.Command{
		Use:   spec.name,
		Short: spec.purpose,
		Long:  spec.purpose,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithWorkflows(cmd, func() error {
				return releaseWorkRun(args)
			})
		},
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeCommandUsage(cmd.OutOrStdout(), spec)
	})
	return c
}

func releaseWorkRun(args []string) error {
	return pipe.StashPipe("release-work", func() error {
		return pipe.BranchPipe("release-work", func() error {
			return releaseWorkLogic(args)
		})
	})
}

func releaseWorkLogic(args []string) error {
	defaultBranch := config.DefaultBranch()
	if err := git.Verbose("checkout", defaultBranch); err != nil {
		return err
	}
	if err := git.Verbose("pull", "--tags"); err != nil {
		return err
	}
	newTag := argAt(args, 0)
	lastTag := state.LastTag()
	if newTag == "" {
		answer, err := readLineAnswer(fmt.Sprintf("'%s' is the last tag. Which one will be next? ", lastTag))
		if err != nil {
			return err
		}
		newTag = answer
	}
	messageFile := "tag-message"
	diapason := lastTag + "...HEAD"
	notesFrom := lastTag
	if lastTag == "" {
		diapason = "HEAD"
		notesFrom = "all-commits"
	}
	var body strings.Builder
	fmt.Fprintf(&body, "Release %s\n\n", newTag)
	logOut := git.OutputOK("log", diapason, "--pretty=format:- %s", "--reverse")
	if logOut != "" {
		body.WriteString(logOut)
		body.WriteString("\n")
	}
	body.WriteString("\n\n")
	if err := os.WriteFile(messageFile, []byte(body.String()), 0o644); err != nil {
		return err
	}
	if err := git.Verbose("tag", "--annotate", "--file", messageFile, "--edit", newTag); err != nil {
		return err
	}
	_ = os.Remove(messageFile)
	if err := git.Verbose("push", "--tags"); err != nil {
		return err
	}
	notes, err := formatReleaseNotes([]string{"smart", notesFrom, "HEAD"})
	if err != nil {
		return err
	}
	copyNotesIfPossible(notes)
	return nil
}
