package git

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/bees-hive/elegant-git/internal/text"
)

// RealRunner invokes the git binary on the host.
type RealRunner struct{}

func (RealRunner) Verbose(args ...string) error {
	text.CommandText(append([]string{"git"}, args...)...)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func (RealRunner) VerboseOp(processor func(string), args ...string) error {
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

func (RealRunner) VerboseOpLines(lineFn func(string), args ...string) error {
	text.CommandText(append([]string{"git"}, args...)...)
	return streamGitLines(true, lineFn, args...)
}

func (RealRunner) StreamLines(lineFn func(string), args ...string) error {
	return streamGitLines(false, lineFn, args...)
}

func streamGitLines(echo bool, lineFn func(string), args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	var wg sync.WaitGroup
	var scanErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(pr)
		for sc.Scan() {
			line := sc.Text()
			if echo {
				fmt.Fprintln(os.Stdout, line)
			}
			if lineFn != nil {
				lineFn(line)
			}
		}
		scanErr = sc.Err()
	}()
	runErr := cmd.Run()
	_ = pw.Close()
	wg.Wait()
	if runErr != nil {
		return runErr
	}
	return scanErr
}

func (RealRunner) Output(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return buf.String(), err
	}
	return strings.TrimSpace(buf.String()), nil
}

func (RealRunner) OutputOK(args ...string) string {
	out, _ := RealRunner{}.Output(args...)
	return strings.TrimSpace(out)
}
