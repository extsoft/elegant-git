package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	completioncmd "github.com/bees-hive/elegant-git/internal/cli/completion"
	gitcmd "github.com/bees-hive/elegant-git/internal/cli/git"
	hookcmd "github.com/bees-hive/elegant-git/internal/cli/hook"
	legacyshim "github.com/bees-hive/elegant-git/internal/cli/legacy"
	memorycmd "github.com/bees-hive/elegant-git/internal/cli/memory"
	profilecmd "github.com/bees-hive/elegant-git/internal/cli/profile"
	releasecmd "github.com/bees-hive/elegant-git/internal/cli/release"
	repocmd "github.com/bees-hive/elegant-git/internal/cli/repo"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	versioncmd "github.com/bees-hive/elegant-git/internal/cli/version"
	workcmd "github.com/bees-hive/elegant-git/internal/cli/work"
	"github.com/bees-hive/elegant-git/internal/deprecation"
	"github.com/bees-hive/elegant-git/internal/exitcode"
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/bees-hive/elegant-git/internal/runtime"
	"github.com/bees-hive/elegant-git/internal/version"
	"github.com/bees-hive/elegant-git/internal/workflows"
	"github.com/spf13/cobra"
)

var nonInteractive bool

var rootCmd = &cobra.Command{
	Use:           "git-elegant",
	Short:         "An assistant who carefully automates routine work with Git.",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version.Version,
	Run: func(cmd *cobra.Command, _ []string) {
		writeRootUsage(cmd.OutOrStdout())
	},
}

// Execute runs the git-elegant CLI.
func Execute() {
	defer deprecation.Flush()
	if err := rootCmd.Execute(); err != nil {
		if isUnknownCommand(err) {
			name := unknownCommandName(err)
			fmt.Fprintf(os.Stderr, "Unknown command: git elegant %s\n", name)
			writeRootUsage(os.Stderr)
			os.Exit(exitcode.UnknownCommand)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeRootUsage(cmd.OutOrStdout())
	})
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	rootCmd.PersistentFlags().BoolVar(&workflows.Skip, "no-workflows", false, "disables available workflows")
	rootCmd.PersistentFlags().BoolVar(&nonInteractive, "non-interactive", false, "disable prompts; fail when required input is missing")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if err := guardInvocationDepth(); err != nil {
			return err
		}
		git.Use(git.RealRunner{})
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		ctx = git.WithRunner(ctx, git.RealRunner{})
		ctx = runtime.WithWorkspace(ctx, runtime.DefaultWorkspace())
		ctx = runtime.WithEditor(ctx, runtime.DefaultEditor())
		ctx = cliruntime.WithStdin(ctx, os.Stdin)
		if nonInteractive || os.Getenv("ELEGANT_GIT_NON_INTERACTIVE") == "1" {
			ctx = prompt.WithPrompter(ctx, prompt.NewNonInteractive())
		} else {
			ctx = prompt.WithPrompter(ctx, prompt.NewTTY(os.Stdin, os.Stdout))
		}
		cmd.SetContext(ctx)
		return nil
	}

	rootCmd.AddCommand(versioncmd.NewCommand())
	rootCmd.AddCommand(completioncmd.NewCommand())

	memoryCmd := memorycmd.NewCommand()
	AttachObjectGroup(memoryCmd, "memory")
	rootCmd.AddCommand(memoryCmd)

	gitCmd := gitcmd.NewCommand()
	AttachObjectGroup(gitCmd, "git")
	rootCmd.AddCommand(gitCmd)

	repoCmd := repocmd.NewCommand()
	AttachObjectGroup(repoCmd, "repo")
	rootCmd.AddCommand(repoCmd)

	profileCmd := profilecmd.NewCommand()
	AttachObjectGroup(profileCmd, "profile")
	rootCmd.AddCommand(profileCmd)

	hookCmd := hookcmd.NewCommand()
	AttachObjectGroup(hookCmd, "hook")
	rootCmd.AddCommand(hookCmd)

	workCmd := workcmd.NewCommand()
	AttachObjectGroup(workCmd, "work")
	rootCmd.AddCommand(workCmd)

	releaseCmd := releasecmd.NewCommand()
	AttachObjectGroup(releaseCmd, "release")
	rootCmd.AddCommand(releaseCmd)
	legacyshim.RegisterShims(rootCmd)
}

func isUnknownCommand(err error) bool {
	return strings.Contains(err.Error(), "unknown command")
}

const invocationDepthEnv = "ELEGANT_GIT_DEPTH"

func guardInvocationDepth() error {
	const maxDepth = 16
	d := 0
	if s := os.Getenv(invocationDepthEnv); s != "" {
		n, err := strconv.Atoi(s)
		if err == nil {
			d = n
		}
	}
	if d >= maxDepth {
		return fmt.Errorf(
			"elegant-git: nested invocation limit (%d); check workflow hooks for recursive `git elegant` / `git deliver-work` calls",
			maxDepth,
		)
	}
	_ = os.Setenv(invocationDepthEnv, strconv.Itoa(d+1))
	return nil
}

func unknownCommandName(err error) string {
	msg := err.Error()
	const prefix = `unknown command "`
	if i := strings.Index(msg, prefix); i >= 0 {
		start := i + len(prefix)
		if j := strings.Index(msg[start:], `"`); j >= 0 {
			return msg[start : start+j]
		}
	}
	return ""
}
