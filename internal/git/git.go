// Package git wraps git invocations with verbose command echoing.
package git

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/bees-hive/elegant-git/internal/text"
)

// Verbose prints and runs git with the given arguments.
func Verbose(args ...string) error {
	text.CommandText(append([]string{"git"}, args...)...)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// VerboseOp prints and runs git, then passes combined stdout/stderr to processor.
func VerboseOp(processor func(string), args ...string) error {
	text.CommandText(append([]string{"git"}, args...)...)
	cmd := exec.Command("git", args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return err
	}
	processor(buf.String())
	return nil
}

// Output runs git quietly and returns combined stdout/stderr.
func Output(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return buf.String(), err
	}
	return strings.TrimSpace(buf.String()), nil
}

// OutputOK runs git and returns stdout even on non-zero exit (like bash 2>&1 || true).
func OutputOK(args ...string) string {
	out, _ := Output(args...)
	return strings.TrimSpace(out)
}
