package statefmt

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/text"
)

// Field is one aligned key/value line inside a list block.
type Field struct {
	Key, Value string
}

// Step is one command in a trailing further-steps section.
type Step struct {
	Command, Comment string
}

// PrintHeading writes a colored section heading (green on a TTY).
func PrintHeading(w io.Writer, indent, heading string) {
	if heading == "" {
		return
	}
	if indent != "" {
		fmt.Fprint(w, indent)
	}
	text.Finfo(w, heading)
}

// PrintBlock writes heading (if any) and indented aligned fields.
func PrintBlock(w io.Writer, indent, heading string, fields []Field) {
	PrintHeading(w, indent, heading)
	PrintFields(w, indent+"  ", fields)
}

// PrintFields writes aligned "key: value" lines at indent.
func PrintFields(w io.Writer, indent string, fields []Field) {
	width := 0
	for _, f := range fields {
		if n := len(f.Key) + 1; n > width {
			width = n
		}
	}
	for _, f := range fields {
		fmt.Fprintf(w, "%s%-*s %s\n", indent, width, f.Key+":", f.Value)
	}
}

// Item is one catalog block: a heading plus aligned fields.
type Item struct {
	Heading string
	Fields  []Field
}

// PrintCatalog writes blank-line-separated heading blocks.
func PrintCatalog(w io.Writer, indent string, items []Item) {
	for i, item := range items {
		if i > 0 {
			fmt.Fprintln(w)
		}
		PrintBlock(w, indent, item.Heading, item.Fields)
	}
}

// PrintFurtherSteps writes a trailing further-steps section. No-op when steps is empty.
func PrintFurtherSteps(w io.Writer, steps []Step) {
	if len(steps) == 0 {
		return
	}
	fmt.Fprintln(w)
	text.Finfo(w, "Further steps:")
	width := 0
	for _, s := range steps {
		if len(s.Command) > width {
			width = len(s.Command)
		}
	}
	for _, s := range steps {
		if s.Comment == "" {
			fmt.Fprintf(w, "  %s\n", s.Command)
			continue
		}
		fmt.Fprintf(w, "  %-*s  %s\n", width, s.Command, s.Comment)
	}
}

// PrintLines writes text as-is, dropping only a trailing newline.
func PrintLines(w io.Writer, text string) {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return
	}
	fmt.Fprintln(w, text)
}

// FileStatusLine returns path, or path with a missing/unreadable suffix.
func FileStatusLine(path string) string {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return path + " (not created yet)"
		}
		return path + " (unreadable)"
	}
	return path
}

// PrintGitIdentity writes user.name, user.email, signing key, gpg program, and editor.
func PrintGitIdentity(w io.Writer, prefix string, get func(string) string) {
	fmt.Fprintf(w, "%suser.name:    %s\n", prefix, text.OrUnset(get("user.name")))
	fmt.Fprintf(w, "%suser.email:   %s\n", prefix, text.OrUnset(get("user.email")))
	fmt.Fprintf(w, "%ssigning key:  %s\n", prefix, text.OrUnset(get("user.signingkey")))
	fmt.Fprintf(w, "%sgpg program:  %s\n", prefix, text.OrUnset(get("gpg.program")))
	fmt.Fprintf(w, "%seditor:       %s\n", prefix, text.OrUnset(get("core.editor")))
}

// ReposWithWorkspace counts registry entries that have a workspace_id.
func ReposWithWorkspace(s *shared.State) int {
	n := 0
	for _, r := range s.Repositories {
		if r != nil && r.WorkspaceID != "" {
			n++
		}
	}
	return n
}

// WorkspaceName returns the display name for workspaceID, or "(none)" / "".
func WorkspaceName(s *shared.State, workspaceID string) string {
	if workspaceID == "" {
		return "(none)"
	}
	if p, err := shared.GetWorkspace(s, workspaceID); err == nil {
		return p.Name
	}
	return ""
}
