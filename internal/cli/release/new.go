package release

import (
	"fmt"
	"os"
	"strings"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/config"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/pipe"
	"github.com/bees-hive/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

var newID = cmdid.ID{Command: "release", Action: "new"}

func newNewCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "new <name>",
		Short: "Releases the default development branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, newID, func() error {
				return newRun(cmd, args)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func newRun(cmd *cobra.Command, args []string) error {
	return pipe.StashPipe(newID, func() error {
		return pipe.BranchPipe(newID, func() error {
			return newLogic(cmd, args)
		})
	})
}

func newLogic(cmd *cobra.Command, args []string) error {
	var tagName string
	if err := argspec.ResolveCmd(cmd, args, argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInput("name", 0, true, "Release tag name", &tagName, nil),
	}}); err != nil {
		return err
	}
	defaultBranch := config.DefaultBranch()
	if err := git.Verbose("checkout", defaultBranch); err != nil {
		return err
	}
	if err := git.Verbose("pull", "--tags"); err != nil {
		return err
	}
	lastTag := state.LastTag()
	newTag := tagName
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
	cliruntime.CopyNotesIfPossible(notes)
	return nil
}
