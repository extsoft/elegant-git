package workspace

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

var runExecInteractiveFn = runExecInteractive

type execStreams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func execSpec(name *string) argspec.Spec {
	in := argspec.PositionalInputWithComplete("name", 0, false, "Workspace name", name, nil, sources.Workspaces, true)
	in.OmitInteractive = true
	return argspec.Spec{Inputs: []argspec.Input{in}}
}

func newExecCommand() *cobra.Command {
	var name string
	spec := execSpec(&name)
	c := &cobra.Command{
		Use:   "exec [name] -- <command> [args...]",
		Short: "Run a command in every repository linked to a workspace",
		Long: "Runs the given command in each repository linked to the workspace. " +
			"Everything after `--` is passed to the child as an argument vector without shell interpretation. " +
			"Use `sh -c` when you need shell syntax. " +
			"Stdin, stdout, and stderr are inherited by each child. " +
			"The command stops at the first non-zero child exit and returns that exit code.",
		RunE: func(cmd *cobra.Command, args []string) error {
			dash := cmd.ArgsLenAtDash()
			if dash < 0 {
				if len(args) == 0 {
					return runExecInteractiveFn(cmd)
				}
				return cliruntime.NewUsageError(cmd, fmt.Errorf("exec requires -- before the command"))
			}
			beforeDash := args[:dash]
			childArgs := args[dash:]
			if len(childArgs) == 0 {
				return cliruntime.NewUsageError(cmd, fmt.Errorf("exec requires a command after --"))
			}
			if len(beforeDash) > 1 {
				return cliruntime.NewUsageError(cmd, fmt.Errorf("accepts at most one workspace name before --, received %d", len(beforeDash)))
			}
			if err := argspec.Resolve(cmd.Context(), prompt.FromContext(cmd.Context()), beforeDash, spec); err != nil {
				return err
			}
			streams := execStreams{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}
			return execRun(streams, name, childArgs)
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	c.Args = func(cmd *cobra.Command, args []string) error {
		d := cmd.ArgsLenAtDash()
		if d < 0 {
			if len(args) > 0 {
				return cliruntime.NewUsageError(cmd, fmt.Errorf("exec requires -- before the command"))
			}
			return nil
		}
		if d > 1 {
			return cliruntime.NewUsageError(cmd, fmt.Errorf("accepts at most one workspace name before --, received %d", d))
		}
		return nil
	}
	return c
}

func execRun(s execStreams, name string, argv []string) error {
	st, err := shared.Load()
	if err != nil {
		return err
	}
	ws, err := resolveFetchWorkspace(st, name)
	if err != nil {
		return err
	}
	if len(ws.LinkedRepos) == 0 {
		fmt.Fprintln(s.Stderr, "No repositories linked to workspace "+ws.Name)
		return nil
	}

	for _, repoID := range ws.LinkedRepos {
		repo, err := shared.GetRepo(st, repoID)
		if err != nil {
			fmt.Fprintf(s.Stderr, "==> %s\n", repoID)
			fmt.Fprintln(s.Stderr, "repository not in registry")
			printExecDoctorHint(s.Stderr, ws.Name)
			return cliruntime.ExitStatus{Code: 1}
		}
		fmt.Fprintf(s.Stderr, "==> %s (%s)\n", repo.Name, repo.CurrentPath)
		if _, err := os.Stat(repo.CurrentPath); err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(s.Stderr, "path missing: %s\n", repo.CurrentPath)
				printExecDoctorHint(s.Stderr, ws.Name)
				return cliruntime.ExitStatus{Code: 1}
			}
			return err
		}
		env := []string{
			"EG_WORKSPACE=" + ws.Name,
			"EG_REPO_NAME=" + repo.Name,
			"EG_REPO_PATH=" + repo.CurrentPath,
		}
		code, err := runChild(argv, repo.CurrentPath, env, s)
		if err != nil {
			fmt.Fprintln(s.Stderr, err.Error())
			return cliruntime.ExitStatus{Code: 1}
		}
		if code != 0 {
			return cliruntime.ExitStatus{Code: code}
		}
	}
	return nil
}

func runExecInteractive(cmd *cobra.Command) error {
	p := prompt.FromContext(cmd.Context())
	if prompt.NonInteractive(p) {
		return cliruntime.NewUsageError(cmd, fmt.Errorf("exec requires -- before the command"))
	}
	line, err := p.String("Command to run", "")
	if err != nil {
		return err
	}
	argv, err := parseExecLine(line)
	if err != nil {
		return cliruntime.NewUsageError(cmd, err)
	}
	streams := execStreams{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}
	return execRun(streams, "", argv)
}

func parseExecLine(line string) ([]string, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("exec requires a command after --")
	}
	var args []string
	var buf strings.Builder
	var quote rune
	quoted := false
	flush := func() {
		if buf.Len() == 0 && !quoted {
			return
		}
		args = append(args, buf.String())
		buf.Reset()
		quoted = false
	}
	for _, r := range line {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				buf.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			quoted = true
		case unicode.IsSpace(r):
			flush()
		default:
			buf.WriteRune(r)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote in command")
	}
	flush()
	if len(args) == 0 {
		return nil, fmt.Errorf("exec requires a command after --")
	}
	return args, nil
}

func printExecDoctorHint(w io.Writer, workspaceName string) {
	fmt.Fprintf(w, "Run `eg workspace doctor %s` to repair missing paths from the workspace side,\n", workspaceName)
	fmt.Fprintln(w, "or `cd` into a relocated clone and run `eg repo doctor`.")
}

func runChild(argv []string, dir string, env []string, s execStreams) (int, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdin = s.Stdin
	cmd.Stdout = s.Stdout
	cmd.Stderr = s.Stderr

	if err := cmd.Start(); err != nil {
		return mapChildExit(argv[0], err)
	}
	stop := watchChildSignals(cmd)
	defer stop()
	err := cmd.Wait()
	if err == nil {
		return 0, nil
	}
	return mapChildExit(argv[0], err)
}

func mapChildExit(prog string, err error) (int, error) {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if code := exitErr.ExitCode(); code >= 0 {
			return code, nil
		}
		if code, ok := signalExitCode(exitErr); ok {
			return code, nil
		}
		return 1, fmt.Errorf("command terminated without an exit code: %w", err)
	}
	if errors.Is(err, exec.ErrNotFound) {
		if pathHasNonExecutable(prog) {
			return 126, nil
		}
		return 127, nil
	}
	if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.ENOEXEC) || errors.Is(err, fs.ErrPermission) {
		return 126, nil
	}
	return 1, fmt.Errorf("cannot execute %q: %w", prog, err)
}

func pathHasNonExecutable(prog string) bool {
	if prog == "" || strings.ContainsRune(prog, os.PathSeparator) || strings.Contains(prog, "/") {
		return false
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		info, err := os.Stat(filepath.Join(dir, prog))
		if err != nil || info.IsDir() {
			continue
		}
		if info.Mode()&0o111 == 0 {
			return true
		}
	}
	return false
}
