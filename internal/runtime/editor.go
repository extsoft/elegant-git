package runtime

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/text"
)

type editorKeyType struct{}

var editorKey = editorKeyType{}

// EditorFunc opens a file in the user's editor.
type EditorFunc func(path string) error

// DefaultEditor uses core.editor from git config.
func DefaultEditor() EditorFunc {
	return func(path string) error {
		editor := strings.TrimSpace(git.OutputOK("config", "core.editor"))
		if editor == "" {
			editor = "vi"
		}
		text.CommandText(append(strings.Fields(editor), path)...)
		cmd := exec.Command("sh", "-c", shellQuote(editor)+" "+shellQuote(path))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// WithEditor stores fn on ctx.
func WithEditor(ctx context.Context, fn EditorFunc) context.Context {
	return context.WithValue(ctx, editorKey, fn)
}

// EditorFromContext returns the editor func from ctx or DefaultEditor.
func EditorFromContext(ctx context.Context) EditorFunc {
	if fn, ok := ctx.Value(editorKey).(EditorFunc); ok && fn != nil {
		return fn
	}
	return DefaultEditor()
}
