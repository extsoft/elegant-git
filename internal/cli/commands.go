package cli

import (
	"fmt"
	"io"
	"sort"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/spf13/cobra"
)

// AllCanonicalCommandIDs returns object.action ids for every registered subcommand.
func AllCanonicalCommandIDs() []string {
	var ids []string
	for _, g := range commandGroups {
		for _, c := range g.commands {
			ids = append(ids, g.object+"."+c.action)
		}
	}
	sort.Strings(ids)
	return ids
}

const siteURL = "https://elegant-git.extsoft.pro"

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
	{object: "memory", title: "inspect elegant-git memory", commands: []subCommandSpec{
		{action: "list", purpose: "Summarizes shared memory paths and counts."},
		{action: "workspaces", purpose: "Lists workspaces or shows one workspace's details."},
		{action: "repositories", purpose: "Lists managed repositories or shows one repository's details."},
	}},
	{object: "git", title: "configure Git installation", commands: []subCommandSpec{
		{action: "configure", purpose: "Configures your Git installation."},
		{action: "list", purpose: "Shows global Git installation and shared memory state."},
		{action: "doctor", purpose: "Diagnoses and repairs your Git installation."},
	}},
	{object: "workspace", title: "manage git workspaces", commands: []subCommandSpec{
		{action: "list", purpose: "Lists workspaces or shows the current or named one."},
		{action: "new", purpose: "Creates a workspace."},
		{action: "link", purpose: "Links the current repository to a workspace."},
		{action: "edit", purpose: "Edits a workspace and optionally applies it to linked repos."},
		{action: "delete", purpose: "Deletes a workspace and unlinks its repositories."},
		{action: "fetch", purpose: "Fetches remotes for repositories linked to a workspace."},
		{action: "doctor", purpose: "Diagnoses and repairs a workspace."},
	}},
	{object: "repo", title: "manage repositories", commands: []subCommandSpec{
		{action: "list", purpose: "Shows repository memory and registry state for the current repository."},
		{action: "init", purpose: "Initializes a new repository and configures it."},
		{action: "clone", purpose: "Clones a remote repository and configures it."},
		{action: "configure", purpose: "Configures the current local Git repository."},
		{action: "sync", purpose: "Re-applies workspace settings to repositories."},
		{action: "prune", purpose: "Removes useless local branches."},
		{action: "doctor", purpose: "Diagnoses and repairs the current repository."},
	}},
	{object: "hook", title: "manage command hooks", commands: []subCommandSpec{
		{action: "list", purpose: "Lists configured hook file paths."},
		{action: "new", purpose: "Creates a new hook file."},
		{action: "edit", purpose: "Opens a hook file in your editor."},
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

// AttachObjectHelp sets HelpFunc so `eg <object> --help` lists actions.
func AttachObjectHelp(c *cobra.Command, object string) {
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeObjectUsage(cmd.OutOrStdout(), object)
	})
}

// AttachObjectGroup sets Run and HelpFunc so `eg <object>` shows that object's subcommands.
func AttachObjectGroup(c *cobra.Command, object string) {
	AttachObjectHelp(c, object)
	c.Run = func(cmd *cobra.Command, _ []string) {
		writeObjectUsage(cmd.OutOrStdout(), object)
	}
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
	fmt.Fprintf(w, "usage: eg %s <action> [-h | --help] [--no-workflows] [args]\n\n", group.object)
	writeAligned(w, "    ", []alignedRow{
		{name: "-h, --help", desc: "displays help for an action"},
		{name: "--no-workflows", desc: "disables available hooks"},
		{name: "--non-interactive", desc: "disables prompts; fails when input is missing"},
	})
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Actions:")
	writeCommandRows(w, "  ", commandNameWidth(group.commands), group.commands)
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
	fmt.Fprintln(w, "usage: eg [-h | --help | --version]")
	fmt.Fprintln(w, "   or: eg <object> <action> [-h | --help] [--no-workflows] [args]")
	fmt.Fprintln(w)
	writeAligned(w, "    ", []alignedRow{
		{name: "-h, --help", desc: "displays help"},
		{name: "--version", desc: "displays program version"},
		{name: "--no-workflows", desc: "disables available hooks"},
		{name: "--non-interactive", desc: "disables prompts; fails when input is missing"},
	})
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Objects:")
	width := actionNameWidth()
	for _, g := range commandGroups {
		fmt.Fprintf(w, "  %s — %s\n", g.object, g.title)
		writeCommandRows(w, "    ", width, g.commands)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Also: version, completion")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Please visit %s to find out more.\n\n", siteURL)
}

type alignedRow struct {
	name string
	desc string
}

func commandNameWidth(commands []subCommandSpec) int {
	n := 0
	for _, c := range commands {
		if len(c.action) > n {
			n = len(c.action)
		}
	}
	return n
}

func actionNameWidth() int {
	n := 0
	for _, g := range commandGroups {
		if w := commandNameWidth(g.commands); w > n {
			n = w
		}
	}
	return n
}

func writeCommandRows(w io.Writer, indent string, width int, commands []subCommandSpec) {
	for _, c := range commands {
		fmt.Fprintf(w, "%s%-*s  %s\n", indent, width, c.action, c.purpose)
	}
}

func writeAligned(w io.Writer, indent string, rows []alignedRow) {
	width := 0
	for _, r := range rows {
		if len(r.name) > width {
			width = len(r.name)
		}
	}
	for _, r := range rows {
		fmt.Fprintf(w, "%s%-*s  %s\n", indent, width, r.name, r.desc)
	}
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
