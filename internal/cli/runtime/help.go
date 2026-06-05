package runtime

import (
	"fmt"
	"io"
	"strings"

	"github.com/bees-hive/elegant-git/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const helpTextWidth = 72

const (
	optionLineIndent = "  "
	optionDescIndent = "      "
)

// CommandHelp prints usage, long description, flags, and version footer.
func CommandHelp(cmd *cobra.Command, _ []string) {
	PrintCommandHelp(cmd.OutOrStdout(), cmd)
}

// PrintCommandHelp writes usage, long description, flags, and version footer.
func PrintCommandHelp(w io.Writer, cmd *cobra.Command) {
	fmt.Fprintf(w, "Usage: %s\n", cmd.UseLine())
	if cmd.Long != "" {
		fmt.Fprintf(w, "\n%s\n", wrapWords(cmd.Long, helpTextWidth))
	} else if cmd.Short != "" {
		fmt.Fprintf(w, "\n%s\n", wrapWords(cmd.Short, helpTextWidth))
	}
	if cmd.HasAvailableLocalFlags() {
		fmt.Fprintln(w, "\nFlags:")
		cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
			printFlagUsage(w, f, helpTextWidth)
		})
	}
	if version.Version != "" {
		fmt.Fprintf(w, "\nVersion: %s\n", version.Version)
	}
}

func printFlagUsage(out io.Writer, f *pflag.Flag, width int) {
	var flagLine string
	if f.Shorthand != "" && f.Name != f.Shorthand {
		flagLine = optionLineIndent + "-" + f.Shorthand + ", --" + f.Name
	} else {
		flagLine = optionLineIndent + "--" + f.Name
	}
	fmt.Fprintln(out, flagLine)
	descCols := width - len(optionDescIndent)
	if descCols < 10 {
		descCols = 10
	}
	for _, line := range strings.Split(wrapWords(f.Usage, descCols), "\n") {
		if line == "" {
			fmt.Fprintln(out)
			continue
		}
		fmt.Fprintf(out, "%s%s\n", optionDescIndent, line)
	}
	fmt.Fprintln(out)
}

func wrapWords(s string, width int) string {
	if width <= 0 {
		return s
	}
	var b strings.Builder
	for _, para := range strings.Split(s, "\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			b.WriteString("\n")
			continue
		}
		for len(para) > width {
			i := strings.LastIndex(para[:width+1], " ")
			if i <= 0 {
				i = width
			}
			b.WriteString(para[:i])
			b.WriteString("\n")
			para = strings.TrimSpace(para[i:])
		}
		b.WriteString(para)
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}
