package workspace

import (
	"fmt"
	"io"
	"os"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/cli/completion"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cli/sources"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/memory/repoid"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

func fetchSpec(name *string) argspec.Spec {
	in := argspec.PositionalInputWithComplete("name", 0, false, "Workspace name", name, nil, sources.Workspaces, true)
	in.OmitInteractive = true
	return argspec.Spec{Inputs: []argspec.Input{in}}
}

func newFetchCommand() *cobra.Command {
	var name string
	spec := fetchSpec(&name)
	c := &cobra.Command{
		Use:   "fetch [name]",
		Short: "Fetch remotes for repositories linked to a workspace",
		Long:  "Runs `git fetch --all --tags --prune` in every repository linked to the workspace. When name is omitted, uses the workspace linked to the current repository. Prune drops stale remote-tracking branches. On a TTY, shows a progress bar, a list of processed repositories, and ephemeral logs for the current fetch. Exits non-zero if any repository fetch fails.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
				return err
			}
			return fetchRun(cmd.OutOrStdout(), name)
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func fetchRun(w io.Writer, name string) error {
	s, err := shared.Load()
	if err != nil {
		return err
	}
	ws, err := resolveFetchWorkspace(s, name)
	if err != nil {
		return err
	}
	if len(ws.LinkedRepos) == 0 {
		fmt.Fprintln(w, "No repositories linked to workspace "+ws.Name)
		return nil
	}

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()

	ui := newFetchDisplay(w, len(ws.LinkedRepos))
	if ui.tty {
		fmt.Fprint(w, "\033[?25l")
		defer fmt.Fprint(w, "\033[?25h")
	}
	fetched, skipped, failed := 0, 0, 0
	for _, repoID := range ws.LinkedRepos {
		repo, err := shared.GetRepo(s, repoID)
		if err != nil {
			ui.start(repoID)
			ui.finish(repoID, "fail", "not in registry")
			failed++
			continue
		}
		ui.start(repo.Name)
		if err := os.Chdir(repo.CurrentPath); err != nil {
			ui.finish(repo.Name, "fail", "path missing: "+repo.CurrentPath)
			failed++
			continue
		}
		if !state.AreThereRemotes() {
			ui.finish(repo.Name, "skip", "no remotes")
			skipped++
			continue
		}
		err = git.StreamLines(ui.log, "fetch", "--all", "--tags", "--prune")
		if err != nil {
			ui.finish(repo.Name, "fail", err.Error())
			failed++
			continue
		}
		ui.finish(repo.Name, "ok", "")
		fetched++
	}
	ui.close()
	fmt.Fprintf(w, "Fetched: %d | Skipped: %d | Failed: %d\n", fetched, skipped, failed)
	if failed > 0 {
		return fmt.Errorf("fetch failed for %d of %d repositories", failed, len(ws.LinkedRepos))
	}
	return nil
}

func resolveFetchWorkspace(s *shared.State, name string) (*shared.Workspace, error) {
	if name != "" {
		_, ws, err := shared.GetWorkspaceByName(s, name)
		return ws, err
	}
	repoID, err := repoid.ReadLocal()
	if err != nil || repoID == "" {
		return nil, fmt.Errorf("not a configured elegant-git repository; run repo configure first")
	}
	reg, err := shared.GetRepo(s, repoID)
	if err != nil {
		return nil, fmt.Errorf("not a configured elegant-git repository; run repo configure first")
	}
	if reg.WorkspaceID == "" {
		return nil, fmt.Errorf("current repository is not linked to a workspace; pass a workspace name or run repo configure")
	}
	return shared.GetWorkspace(s, reg.WorkspaceID)
}
