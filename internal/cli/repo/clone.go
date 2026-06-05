package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/cli/completion"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cmdid"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var cloneID = cmdid.ID{Command: "repo", Action: "clone"}

func cloneSpec(repository, profile, directory *string) argspec.Spec {
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInput("repository", 0, true, "Repository URL or path", repository, nil),
		profileInput(1, profile),
		argspec.PositionalInput("directory", 2, false, "Target directory", directory, nil),
	}}
}

func newCloneCommand() *cobra.Command {
	var repository, profileName, directory string
	spec := cloneSpec(&repository, &profileName, &directory)
	c := &cobra.Command{
		Use:   "clone <repository> <profile> [<directory>]",
		Short: "Clones a remote repository and configures it",
		Long:  "Runs git clone then repo configure.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithCompat(cmd, cloneID, "acquire-repository", func() error {
				if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
					return err
				}
				return cloneRun(cmd, repository, profileName, directory)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func cloneRun(cmd *cobra.Command, repository, profile, directory string) error {
	cloneArgs := []string{repository}
	if directory != "" {
		cloneArgs = append(cloneArgs, directory)
	}
	if err := git.Verbose(append([]string{"clone"}, cloneArgs...)...); err != nil {
		return err
	}
	location := cloneTargetDir(cloneArgs)
	text.InfoText(fmt.Sprintf("The repository was cloned into '%s' directory.", location))
	if err := os.Chdir(location); err != nil {
		return err
	}
	return ConfigureRun(cmd, profile)
}

func cloneTargetDir(args []string) string {
	location := args[len(args)-1]
	if strings.HasSuffix(location, ".git") {
		base := filepath.Base(location)
		location = strings.TrimSuffix(base, ".git")
	}
	return location
}
