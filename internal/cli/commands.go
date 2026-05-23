package cli

import (
	"fmt"
	"io"

	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/spf13/cobra"
)

const siteURL = "https://elegant-git.bees-hive.org"

type commandGroup struct {
	object   string
	title    string
	commands []subCommandSpec
}

type subCommandSpec struct {
	action  string
	purpose string
}

var commandGroups = []commandGroup{
	{object: "git", title: "configure Git installation", commands: []subCommandSpec{
		{action: "configure", purpose: "Configures your Git installation."},
		{action: "migrate", purpose: "Migrates global aliases and acquired marker."},
	}},
	{object: "repo", title: "manage repositories", commands: []subCommandSpec{
		{action: "configure", purpose: "Configures the current local Git repository."},
		{action: "clone", purpose: "Clones a remote repository and configures it."},
		{action: "init", purpose: "Initializes a new repository and configures it."},
		{action: "prune", purpose: "Removes useless local branches."},
		{action: "migrate", purpose: "Migrates local aliases, hooks, and pipe keys."},
	}},
	{object: "hook", title: "manage command hooks", commands: []subCommandSpec{
		{action: "list", purpose: "Lists configured hook file paths."},
		{action: "new", purpose: "Creates a new hook file."},
		{action: "edit", purpose: "Opens a hook file in your editor."},
		{action: "migrate", purpose: "Migrates repo-tracked hooks to the new layout."},
	}},
	{object: "work", title: "day-to-day contributions", commands: []subCommandSpec{
		{action: "start", purpose: "Creates a new branch."},
		{action: "save", purpose: "Commits current modifications."},
		{action: "amend", purpose: "Amends the last commit."},
		{action: "list", purpose: "Prints HEAD state."},
		{action: "polish", purpose: "Rebases HEAD interactively."},
		{action: "sync", purpose: "Actualizes the branch with upstream commits."},
		{action: "push", purpose: "Publishes HEAD to a remote repository."},
		{action: "track", purpose: "Checks out a remote-tracking branch."},
		{action: "accept", purpose: "Adds modifications to the default development branch."},
	}},
	{object: "release", title: "manage releases", commands: []subCommandSpec{
		{action: "new", purpose: "Releases the default development branch."},
		{action: "notes", purpose: "Prints a release log between two refs."},
	}},
}

// AttachObjectGroup sets Run and HelpFunc so `git elegant <object>` shows that object's subcommands.
func AttachObjectGroup(c *cobra.Command, object string) {
	c.Run = func(cmd *cobra.Command, _ []string) {
		writeObjectUsage(cmd.OutOrStdout(), object)
	}
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeObjectUsage(cmd.OutOrStdout(), object)
	})
}

func writeObjectUsage(w io.Writer, object string) {
	var group *commandGroup
	for i := range commandGroups {
		if commandGroups[i].object == object {
			group = &commandGroups[i]
			break
		}
	}
	if group == nil {
		writeRootUsage(w)
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s — %s\n\n", group.object, group.title)
	fmt.Fprintf(w, "usage: git elegant %s <action> [-h | --help] [--no-workflows] [args]\n\n", group.object)
	fmt.Fprintln(w, "    -h, --help       displays help for an action")
	fmt.Fprintln(w, "    --no-workflows   disables available hooks")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Actions:")
	maxLen := 0
	for _, c := range group.commands {
		if len(c.action) > maxLen {
			maxLen = len(c.action)
		}
	}
	for _, c := range group.commands {
		fmt.Fprintf(w, "  %-*s  %s\n", maxLen, c.action, c.purpose)
	}
	if object == "hook" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Hooks live under:")
		fmt.Fprintln(w, "  <repo>/.config/elegant-git/hooks/<command>-<action>-{ahead,after}   (repo-tracked)")
		fmt.Fprintln(w, "  <repo>/.git/.config/elegant-git/hooks/<command>-<action>-{ahead,after}   (personal)")
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Please visit %s to find out more.\n\n", siteURL)
}

func writeRootUsage(w io.Writer) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "An assistant who carefully automates routine work with Git.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "usage: git elegant [-h | --help | --version]")
	fmt.Fprintln(w, "   or: git elegant <object> <action> [-h | --help] [--no-workflows] [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "    -h, --help       displays help")
	fmt.Fprintln(w, "    --version        displays program version")
	fmt.Fprintln(w, "    --no-workflows   disables available hooks")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Objects:")
	for _, g := range commandGroups {
		fmt.Fprintf(w, "  %s — %s\n", g.object, g.title)
		for _, c := range g.commands {
			fmt.Fprintf(w, "    %-12s %s\n", g.object+" "+c.action, c.purpose)
		}
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Also: version, completion")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Please visit %s to find out more.\n\n", siteURL)
}

func newBaseCommand(use, short, long string) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		Long:          long,
		SilenceUsage:  false,
		SilenceErrors: true,
	}
}

func attachHelp(c *cobra.Command) {
	c.SetHelpFunc(cliruntime.CommandHelp)
}
