package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var cloneID = cmdid.ID{Command: "repo", Action: "clone"}

func newCloneCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "clone <repository> [<directory>]",
		Short: "Clones a remote repository and configures it",
		Long:  "Runs git clone then repo configure.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithCompat(cmd, cloneID, "acquire-repository", func() error {
				return cloneRun(cmd, args)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func cloneRun(cmd *cobra.Command, args []string) error {
	if err := git.Verbose(append([]string{"clone"}, args...)...); err != nil {
		return err
	}
	location := cloneTargetDir(args)
	text.InfoText(fmt.Sprintf("The repository was cloned into '%s' directory.", location))
	if err := os.Chdir(location); err != nil {
		return err
	}
	return ConfigureRun(cmd)
}

func cloneTargetDir(args []string) string {
	location := args[len(args)-1]
	if strings.HasSuffix(location, ".git") {
		base := filepath.Base(location)
		location = strings.TrimSuffix(base, ".git")
	}
	return location
}
