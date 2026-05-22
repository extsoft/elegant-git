package cli

import (
	"fmt"
	"io"
)

func writeRootUsage(w io.Writer) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "An assistant who carefully automates routine work with Git.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "usage: git elegant [-h | --help | help | --version | version]")
	fmt.Fprintln(w, "   or: git elegant <command> [-h | --help | help]")
	fmt.Fprintln(w, "   or: git elegant <command> [--no-workflows] [args]")
	fmt.Fprintln(w, "   or: git elegant [--no-workflows] <command> [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "    -h, --help, help    displays help")
	fmt.Fprintln(w, "    --version, version  displays program version")
	fmt.Fprintln(w, "    --no-workflows      disables available workflows")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "There are commands used in various situations such as")
	for _, g := range commandGroups {
		fmt.Fprintln(w, g.title)
		for _, c := range g.commands {
			fmt.Fprintf(w, "    %-20s %s\n", c.name, c.purpose)
		}
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Please visit %s to find out more.\n", siteURL)
	fmt.Fprintln(w)
}

func writeCommandUsage(w io.Writer, spec commandSpec) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, spec.synopsis)
	fmt.Fprintln(w)
	if spec.description != "" {
		fmt.Fprintln(w, spec.description)
		fmt.Fprintln(w)
	}
}
