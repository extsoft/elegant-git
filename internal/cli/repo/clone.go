package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var cloneID = cmdid.ID{Command: "repo", Action: "clone"}

func cloneOptionalWorkspaceInput(index int, workspace *string) argspec.Input {
	in := argspec.PositionalInputWithComplete(
		"workspace", index, false, "Workspace name", workspace, nil, sources.WorkspacesWithCreateNew, false,
	)
	in.OmitInteractive = true
	return in
}

func cloneSpec(repository, workspace, directory *string) argspec.Spec {
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInput("repository", 0, true, "Repository URL or path", repository, nil),
		cloneOptionalWorkspaceInput(1, workspace),
		argspec.PositionalInput("directory", 2, false, "Target directory", directory, func() string {
			return defaultCloneDir(*repository)
		}),
	}}
}

func newCloneCommand() *cobra.Command {
	var repository, workspaceName, directory string
	spec := cloneSpec(&repository, &workspaceName, &directory)
	c := &cobra.Command{
		Use:   "clone <repository> [<workspace>] [<directory>]",
		Short: "Clones a remote repository and configures it",
		Long:  "Runs git clone then repo configure. When workspace is omitted, suggests one from the repository namespace (<domain>/<owner>).",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithCompat(cmd, cloneID, "acquire-repository", func() error {
				if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
					return err
				}
				disambiguateCloneArgs(&workspaceName, &directory)
				return cloneRun(cmd, repository, workspaceName, directory)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func cloneRun(cmd *cobra.Command, repository, workspace, directory string) error {
	if directory == "" {
		directory = defaultCloneDir(repository)
	}
	if err := git.Verbose("clone", repository, directory); err != nil {
		return err
	}
	text.InfoText(fmt.Sprintf("The repository was cloned into '%s' directory.", directory))
	if err := os.Chdir(directory); err != nil {
		return err
	}
	return ConfigureRun(cmd, workspace)
}

// disambiguateCloneArgs moves a second positional into directory when it is
// not a known workspace name (so `clone <url> <dir>` works).
func disambiguateCloneArgs(workspace, directory *string) {
	if *workspace == "" || *directory != "" {
		return
	}
	if *workspace == sources.WorkspaceCreateNew {
		return
	}
	if s, err := shared.Load(); err == nil {
		if _, _, err := shared.GetWorkspaceByName(s, *workspace); err == nil {
			return
		}
	}
	*directory = *workspace
	*workspace = ""
}

// defaultCloneDir mirrors git clone's default destination: last path segment, without .git.
func defaultCloneDir(repository string) string {
	base := filepath.Base(strings.TrimRight(repository, "/"))
	return strings.TrimSuffix(base, ".git")
}
