package cli

type commandSpec struct {
	name        string
	purpose     string
	synopsis    string
	description string
}

type commandGroup struct {
	title    string
	commands []commandSpec
}

const siteURL = "https://elegant-git.bees-hive.org"

var commandGroups = []commandGroup{
	{
		title: " enable Elegnat Git services",
		commands: []commandSpec{
			{name: "acquire-git", purpose: "Configures your Git installation.", synopsis: "usage: git elegant acquire-git"},
			{name: "acquire-repository", purpose: "Configures the current local Git repository.", synopsis: "usage: git elegant acquire-repository"},
			{name: "clone-repository", purpose: "Clones a remote repository and configures it.", synopsis: "usage: git elegant clone-repository [options] <repository> [<directory>]"},
			{name: "init-repository", purpose: "Initializes a new repository and configures it.", synopsis: "usage: git elegant init-repository"},
		},
	},
	{
		title: " serve a repository",
		commands: []commandSpec{
			{name: "prune-repository", purpose: "Removes useless local branches.", synopsis: "usage: git elegant prune-repository"},
		},
	},
	{
		title: " enhance contribution rules",
		commands: []commandSpec{
			{name: "show-workflows", purpose: "Prints file locations of the configured workflows.", synopsis: "usage: git elegant show-workflows"},
			{name: "make-workflow", purpose: "Makes a new workflow file.", synopsis: "usage: git elegant make-workflow <command> <type> <location>"},
			{name: "polish-workflow", purpose: "Opens a given workflow file.", synopsis: "usage: git elegant polish-workflow <file path>"},
		},
	},
	{
		title: " make day-to-day contributions",
		commands: []commandSpec{
			{name: "start-work", purpose: "Creates a new branch.", synopsis: "usage: git elegant start-work <name> [from-ref]"},
			{name: "save-work", purpose: "Commits current modifications.", synopsis: "usage: git elegant save-work"},
			{name: "amend-work", purpose: "Amends the last commit.", synopsis: "usage: git elegant amend-work"},
			{name: "show-work", purpose: "Prints HEAD state.", synopsis: "usage: git elegant show-work"},
			{name: "polish-work", purpose: "Rebases HEAD interactively.", synopsis: "usage: git elegant polish-work"},
			{name: "actualize-work", purpose: "Actualizes the current branch with upstream commits.", synopsis: "usage: git elegant actualize-work [branch-name]"},
		},
	},
	{
		title: " interact with others",
		commands: []commandSpec{
			{name: "deliver-work", purpose: "Publishes HEAD to a remote repository.", synopsis: "usage: git elegant deliver-work [branch-name]"},
			{name: "obtain-work", purpose: "Checkouts a remote-tracking branch.", synopsis: "usage: git elegant obtain-work <name> [local branch]"},
		},
	},
	{
		title: " manage contributions",
		commands: []commandSpec{
			{name: "accept-work", purpose: "Adds modifications to the default development branch.", synopsis: "usage: git elegant accept-work <branch>"},
			{name: "release-work", purpose: "Releases the default development branch.", synopsis: "usage: git elegant release-work [name]"},
			{name: "show-release-notes", purpose: "Prints a release log between two refs.", synopsis: "usage: git elegant show-release-notes [<layout>] [<from-ref>] [<to-ref>]"},
		},
	},
	{
		title: " and others",
		commands: []commandSpec{
			{name: "show-commands", purpose: "Prints Elegant Git commands.", synopsis: "usage: git elegant show-commands"},
		},
	},
}

func allCommandSpecs() []commandSpec {
	var out []commandSpec
	for _, g := range commandGroups {
		out = append(out, g.commands...)
	}
	return out
}

// CommandNames returns all elegant-git subcommand names.
func CommandNames() []string {
	specs := allCommandSpecs()
	names := make([]string, len(specs))
	for i, s := range specs {
		names[i] = s.name
	}
	return names
}

func lookupCommand(name string) (commandSpec, bool) {
	for _, spec := range allCommandSpecs() {
		if spec.name == name {
			return spec, true
		}
	}
	return commandSpec{}, false
}
