// Package clitest helps wire the CLI for integration tests.
package clitest

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/runtime"
	"github.com/spf13/cobra"
)

// Options configures a test root command.
type Options struct {
	Runner git.Runner
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Editor runtime.EditorFunc
}

// ExecuteRoot runs args against a cobra root built by the caller.
func ExecuteRoot(t *testing.T, root *cobra.Command, opts Options, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	deprecation.Reset()
	if opts.Runner != nil {
		git.Use(opts.Runner)
	}
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}
	out := io.Writer(stdoutBuf)
	errW := io.Writer(stderrBuf)
	if opts.Stdout != nil {
		out = opts.Stdout
	}
	if opts.Stderr != nil {
		errW = opts.Stderr
	}
	root.SetOut(out)
	root.SetErr(errW)
	root.SetArgs(args)
	ctx := context.Background()
	ctx = git.WithRunner(ctx, opts.Runner)
	if opts.Runner == nil {
		ctx = git.WithRunner(ctx, git.RealRunner{})
	}
	ctx = runtime.WithRepoLayout(ctx, runtime.DefaultRepoLayout())
	if opts.Editor != nil {
		ctx = runtime.WithEditor(ctx, opts.Editor)
	} else {
		ctx = runtime.WithEditor(ctx, func(string) error { return nil })
	}
	stdin := opts.Stdin
	if stdin == nil {
		stdin = stringsReader("")
	}
	ctx = cliruntime.WithStdin(ctx, stdin)
	root.SetContext(ctx)
	err = root.Execute()
	deprecation.Flush()
	return stdoutBuf.String(), stderrBuf.String(), err
}

func stringsReader(s string) io.Reader {
	return bytes.NewBufferString(s)
}

// AssertCallsContains fails if none of the runner calls match args prefix.
func AssertCallsContains(t *testing.T, mem *git.MemoryRunner, argsPrefix ...string) {
	t.Helper()
	for _, call := range mem.Calls {
		if len(call.Args) >= len(argsPrefix) {
			match := true
			for i, a := range argsPrefix {
				if call.Args[i] != a {
					match = false
					break
				}
			}
			if match {
				return
			}
		}
	}
	t.Fatalf("no call matching %v in %+v", argsPrefix, mem.Calls)
}

var _ = os.Stderr
