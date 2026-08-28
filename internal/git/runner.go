package git

import "context"

type runnerKeyType struct{}

var runnerKey = runnerKeyType{}

// Runner executes git commands (real subprocess or in-memory simulation).
type Runner interface {
	Verbose(args ...string) error
	VerboseOp(processor func(string), args ...string) error
	VerboseOpLines(lineFn func(string), args ...string) error
	StreamLines(lineFn func(string), args ...string) error
	Output(args ...string) (string, error)
	OutputOK(args ...string) string
}

// current is the process-wide runner; production uses RealRunner, tests swap MemoryRunner.
var current Runner = RealRunner{}

// Use sets the process-wide runner (tests and PersistentPreRunE).
func Use(r Runner) {
	current = r
}

// WithRunner stores r on ctx for cobra commands.
func WithRunner(ctx context.Context, r Runner) context.Context {
	return context.WithValue(ctx, runnerKey, r)
}

// FromContext returns the runner from ctx, else the process-wide runner.
func FromContext(ctx context.Context) Runner {
	if r, ok := ctx.Value(runnerKey).(Runner); ok && r != nil {
		return r
	}
	return current
}

// Verbose prints and runs git with the given arguments.
func Verbose(args ...string) error {
	return current.Verbose(args...)
}

// VerboseOp prints and runs git, then passes combined stdout/stderr to processor.
func VerboseOp(processor func(string), args ...string) error {
	return current.VerboseOp(processor, args...)
}

// VerboseOpLines prints git output line-by-line and calls lineFn per line (no full-buffer capture).
func VerboseOpLines(lineFn func(string), args ...string) error {
	return current.VerboseOpLines(lineFn, args...)
}

// StreamLines runs git without printing CommandText or stdout; each output line is passed to lineFn.
func StreamLines(lineFn func(string), args ...string) error {
	return current.StreamLines(lineFn, args...)
}

// Output runs git quietly and returns combined stdout/stderr.
func Output(args ...string) (string, error) {
	return current.Output(args...)
}

// OutputOK runs git and returns stdout even on non-zero exit.
func OutputOK(args ...string) string {
	return current.OutputOK(args...)
}
