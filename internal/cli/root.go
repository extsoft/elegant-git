package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	completioncmd "github.com/extsoft/elegant-git/internal/cli/completion"
	gitcmd "github.com/extsoft/elegant-git/internal/cli/git"
	hookcmd "github.com/extsoft/elegant-git/internal/cli/hook"
	legacyshim "github.com/extsoft/elegant-git/internal/cli/legacy"
	memorycmd "github.com/extsoft/elegant-git/internal/cli/memory"
	releasecmd "github.com/extsoft/elegant-git/internal/cli/release"
	repocmd "github.com/extsoft/elegant-git/internal/cli/repo"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	versioncmd "github.com/extsoft/elegant-git/internal/cli/version"
	workcmd "github.com/extsoft/elegant-git/internal/cli/work"
	workspacecmd "github.com/extsoft/elegant-git/internal/cli/workspace"
	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/extsoft/elegant-git/internal/exitcode"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/runtime"
	"github.com/extsoft/elegant-git/internal/version"
	"github.com/extsoft/elegant-git/internal/workflows"
	"github.com/spf13/cobra"
)

var (
	nonInteractive   bool
	forceInteractive bool
)

var rootCmd = &cobra.Command{
	Use:           "eg",
	Short:         "An assistant who carefully automates routine work with Git.",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version.Version,
	Run: func(cmd *cobra.Command, _ []string) {
		writeRootUsage(cmd.OutOrStdout())
	},
}

// Execute runs the eg CLI.
func Execute() {
	defer deprecation.Flush()
	if err := rootCmd.Execute(); err != nil {
		if isUnknownCommand(err) {
			name := unknownCommandName(err)
			fmt.Fprintf(os.Stderr, "Unknown command: eg %s\n", name)
			writeRootUsage(os.Stderr)
			os.Exit(exitcode.UnknownCommand)
		}
		var ue *cliruntime.UsageError
		if errors.As(err, &ue) {
			fmt.Fprintln(os.Stderr, err.Error())
			cliruntime.EmitCommandHelp(os.Stderr, ue.Cmd)
			os.Exit(exitcode.Usage)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func init() {
	sources.SetHookCommandIDsProvider(AllCanonicalCommandIDs)
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		writeRootUsage(cmd.OutOrStdout())
	})
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	rootCmd.PersistentFlags().BoolVar(&workflows.Skip, "no-workflows", false, "disables available workflows")
	rootCmd.PersistentFlags().BoolVar(&nonInteractive, "non-interactive", false, "disable prompts; fail when required input is missing")
	rootCmd.PersistentFlags().BoolVar(&forceInteractive, "interactive", false, "force prompts on a TTY (overrides CI and --non-interactive)")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		recordDeprecatedSurface(cmd)
		if err := guardInvocationDepth(); err != nil {
			return err
		}
		git.Use(git.RealRunner{})
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		ctx = git.WithRunner(ctx, git.RealRunner{})
		ctx = runtime.WithRepoLayout(ctx, runtime.DefaultRepoLayout())
		ctx = runtime.WithEditor(ctx, runtime.DefaultEditor())
		stdin := cliruntime.StdinFromContext(ctx)
		if stdin == os.Stdin {
			stdin = os.Stdin
		}
		ctx = cliruntime.WithStdin(ctx, stdin)
		mode := cliruntime.ResolveMode(cliruntime.ModeConfig{
			ForceInteractive:    forceInteractive,
			ForceNonInteractive: nonInteractive,
			Stdin:               stdin,
		})
		ctx = prompt.WithPrompter(ctx, cliruntime.PrompterForMode(mode, stdin, os.Stdout))
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

	workspaceCmd := workspacecmd.NewCommand()
	AttachObjectGroup(workspaceCmd, "workspace")
	rootCmd.AddCommand(workspaceCmd)

	legacyProfileCmd := workspacecmd.NewLegacyProfileCommand()
	AttachObjectGroup(legacyProfileCmd, "profile")
	rootCmd.AddCommand(legacyProfileCmd)

	hookCmd := hookcmd.NewCommand()
	AttachObjectGroup(hookCmd, "hook")
	rootCmd.AddCommand(hookCmd)

	workCmd := workcmd.NewCommand()
	AttachObjectHelp(workCmd, "work")
	rootCmd.AddCommand(workCmd)

	releaseCmd := releasecmd.NewCommand()
	AttachObjectGroup(releaseCmd, "release")
	rootCmd.AddCommand(releaseCmd)
	legacyshim.RegisterShims(rootCmd)
	cliruntime.ConfigureCommandTree(rootCmd)
}

func isUnknownCommand(err error) bool {
	return strings.Contains(err.Error(), "unknown command")
}

func recordDeprecatedSurface(cmd *cobra.Command) {
	replacements := map[string]string{
		"memory profiles":  "memory workspaces",
		"workspace create": "workspace new",
	}
	for c := cmd; c != nil; c = c.Parent() {
		surface := c.Annotations[deprecation.SurfaceAnnotation]
		if surface == "" {
			continue
		}
		replacement, ok := replacements[surface]
		if !ok {
			replacement = "workspace"
			if strings.HasPrefix(surface, "profile") {
				replacement = strings.Replace(surface, "profile", "workspace", 1)
			}
		}
		deprecation.RecordRenamedSurface(surface, replacement, "eg repo migrate")
		return
	}
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
			"elegant-git: nested invocation limit (%d); check workflow hooks for recursive `eg` / `git deliver-work` calls",
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
