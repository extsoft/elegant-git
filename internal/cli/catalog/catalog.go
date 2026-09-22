package catalog

import (
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"
)

const SiteURL = "https://elegant-git.extsoft.pro"

// Command is one action under an object.
type Command struct {
	Action  string
	Purpose string
}

// Group is one CLI object and its actions.
type Group struct {
	Object   string
	Title    string
	Commands []Command
}

// Groups is the canonical object/action catalog.
var Groups = []Group{
	{Object: "self", Title: "set up and inspect Elegant Git", Commands: []Command{
		{Action: "configure", Purpose: "Configures your Git installation."},
		{Action: "list", Purpose: "Shows Elegant Git and global Git installation state."},
		{Action: "doctor", Purpose: "Diagnoses and repairs your Git installation."},
	}},
	{Object: "workspace", Title: "manage git workspaces", Commands: []Command{
		{Action: "list", Purpose: "Lists workspaces or shows the current or named one."},
		{Action: "new", Purpose: "Creates a workspace."},
		{Action: "link", Purpose: "Links the current repository to a workspace."},
		{Action: "edit", Purpose: "Edits a workspace and optionally applies it to linked repos."},
		{Action: "delete", Purpose: "Deletes a workspace and unlinks its repositories."},
		{Action: "fetch", Purpose: "Fetches remotes for repositories linked to a workspace."},
		{Action: "doctor", Purpose: "Diagnoses and repairs a workspace."},
	}},
	{Object: "repo", Title: "manage repositories", Commands: []Command{
		{Action: "list", Purpose: "Lists repositories or shows the current or named one."},
		{Action: "init", Purpose: "Initializes a new repository and configures it."},
		{Action: "clone", Purpose: "Clones a remote repository and configures it."},
		{Action: "configure", Purpose: "Configures the current local Git repository."},
		{Action: "sync", Purpose: "Re-applies workspace settings to repositories."},
		{Action: "prune", Purpose: "Removes useless local branches."},
		{Action: "doctor", Purpose: "Diagnoses and repairs the current repository."},
	}},
	{Object: "hook", Title: "manage command hooks", Commands: []Command{
		{Action: "list", Purpose: "Lists configured hook file paths."},
		{Action: "new", Purpose: "Creates a new hook file."},
		{Action: "edit", Purpose: "Opens a hook file in your editor."},
	}},
	{Object: "work", Title: "day-to-day contributions", Commands: []Command{
		{Action: "start", Purpose: "Creates a new branch."},
		{Action: "save", Purpose: "Commits current modifications."},
		{Action: "list", Purpose: "Prints HEAD state."},
		{Action: "polish", Purpose: "Rebases HEAD interactively."},
		{Action: "sync", Purpose: "Actualizes the branch with upstream commits."},
		{Action: "push", Purpose: "Publishes HEAD to a remote repository."},
		{Action: "track", Purpose: "Checks out a remote-tracking branch."},
		{Action: "accept", Purpose: "Adds modifications to the default development branch."},
	}},
	{Object: "release", Title: "manage releases", Commands: []Command{
		{Action: "new", Purpose: "Releases the default development branch."},
		{Action: "notes", Purpose: "Prints a release log between two refs."},
	}},
}

// AllCanonicalCommandIDs returns object.action ids for every registered subcommand.
func AllCanonicalCommandIDs() []string {
	var ids []string
	for _, g := range Groups {
		for _, c := range g.Commands {
			ids = append(ids, g.Object+"."+c.Action)
		}
	}
	sort.Strings(ids)
	return ids
}

// Purpose returns the picker description for an object action, including help and quit.
func Purpose(object, action string) string {
	switch action {
	case "help":
		return "Shows available actions."
	case "quit":
		return "Leave without another action."
	}
	for _, g := range Groups {
		if g.Object == object {
			for _, c := range g.Commands {
				if c.Action == action {
					return c.Purpose
				}
			}
		}
	}
	return ""
}

// AttachObjectHelp sets HelpFunc so `eg <object> --help` lists actions.
func AttachObjectHelp(c *cobra.Command, object string) {
	c.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		WriteObjectUsage(cmd.OutOrStdout(), object)
	})
}

// AttachObjectGroup sets Run and HelpFunc so `eg <object>` shows that object's subcommands.
func AttachObjectGroup(c *cobra.Command, object string) {
	AttachObjectHelp(c, object)
	c.Run = func(cmd *cobra.Command, _ []string) {
		WriteObjectUsage(cmd.OutOrStdout(), object)
	}
}

// WriteObjectUsage prints help for one object command group.
func WriteObjectUsage(w io.Writer, object string) {
	var group *Group
	for i := range Groups {
		if Groups[i].Object == object {
			group = &Groups[i]
			break
		}
	}
	if group == nil {
		WriteRootUsage(w)
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s — %s\n\n", group.Object, group.Title)
	fmt.Fprintf(w, "usage: eg %s <action> [-h | --help] [--no-workflows] [args]\n\n", group.Object)
	writeAligned(w, "    ", []alignedRow{
		{name: "-h, --help", desc: "displays help for an action"},
		{name: "--no-workflows", desc: "disables available hooks"},
		{name: "--non-interactive", desc: "disables prompts; fails when input is missing"},
	})
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Actions:")
	writeCommandRows(w, "  ", commandNameWidth(group.Commands), group.Commands)
	if object == "hook" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Hooks live under:")
		fmt.Fprintln(w, "  <repo>/.config/elegant-git/hooks/<command>-<action>-{ahead,after}   (repo-tracked)")
		fmt.Fprintln(w, "  <repo>/.git/.config/elegant-git/hooks/<command>-<action>-{ahead,after}   (personal)")
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Please visit %s to find out more.\n\n", SiteURL)
}

// WriteRootUsage prints top-level eg help.
func WriteRootUsage(w io.Writer) {
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
	for _, g := range Groups {
		fmt.Fprintf(w, "  %s — %s\n", g.Object, g.Title)
		writeCommandRows(w, "    ", width, g.Commands)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Also: version, completion")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Please visit %s to find out more.\n\n", SiteURL)
}

type alignedRow struct {
	name string
	desc string
}

func commandNameWidth(commands []Command) int {
	n := 0
	for _, c := range commands {
		if len(c.Action) > n {
			n = len(c.Action)
		}
	}
	return n
}

func actionNameWidth() int {
	n := 0
	for _, g := range Groups {
		if w := commandNameWidth(g.Commands); w > n {
			n = w
		}
	}
	return n
}

func writeCommandRows(w io.Writer, indent string, width int, commands []Command) {
	for _, c := range commands {
		fmt.Fprintf(w, "%s%-*s  %s\n", indent, width, c.Action, c.Purpose)
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
