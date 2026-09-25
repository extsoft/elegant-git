package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	clicatalog "github.com/extsoft/elegant-git/internal/cli/catalog"
	completioncmd "github.com/extsoft/elegant-git/internal/cli/completion"
	hookcmd "github.com/extsoft/elegant-git/internal/cli/hook"
	legacyshim "github.com/extsoft/elegant-git/internal/cli/legacy"
	releasecmd "github.com/extsoft/elegant-git/internal/cli/release"
	repocmd "github.com/extsoft/elegant-git/internal/cli/repo"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	selfcmd "github.com/extsoft/elegant-git/internal/cli/self"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	versioncmd "github.com/extsoft/elegant-git/internal/cli/version"
	workcmd "github.com/extsoft/elegant-git/internal/cli/work"
	workspacecmd "github.com/extsoft/elegant-git/internal/cli/workspace"
	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/extsoft/elegant-git/internal/exitcode"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/migrate"
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
	RunE:          runRoot,
}

func runRoot(cmd *cobra.Command, _ []string) error {
	p := prompt.FromContext(cmd.Context())
	if prompt.NonInteractive(p) {
		clicatalog.WriteRootUsage(cmd.OutOrStdout())
		return nil
	}
	inside, err := inGitRepo()
	if err != nil {
		return err
	}
	if !inside {
		clicatalog.WriteRootUsage(cmd.OutOrStdout())
		return nil
	}
	hasCommit, err := headResolves()
	if err != nil {
		return err
	}
	if !hasCommit {
		clicatalog.WriteRootUsage(cmd.OutOrStdout())
		return nil
	}
	detected, err := runDetectedWork(cmd)
	if err != nil || !detected.Ask {
		return err
	}
	choices := rootChoices(detected.Actions)
	ans, err := cliruntime.PickActionOrHelp(cmd, p, repoPickerLabel(), choices, pickerDefault(choices, detected.Ran))
	if err != nil {
		return err
	}
	if ans == "" || ans == "quit" {
		return nil
	}
	object, action, _ := strings.Cut(ans, " ")
	var args []string
	if object == "work" && action == "accept" {
		args = detected.Accept
	}
	return dispatchRoot(cmd, object, action, args)
}

func inGitRepo() (bool, error) {
	out, err := git.OutputC("rev-parse", "--is-inside-work-tree")
	out = strings.TrimSpace(out)
	if err == nil {
		return out == "true", nil
	}
	msg := strings.TrimSpace(out + "\n" + err.Error())
	if strings.Contains(msg, "not a git repository") {
		return false, nil
	}
	out, err = git.Output("rev-parse", "--is-inside-work-tree")
	out = strings.TrimSpace(out)
	if err == nil {
		return out == "true", nil
	}
	if out != "" {
		return false, errors.New(out)
	}
	return false, err
}

func headResolves() (bool, error) {
	if _, err := git.OutputC("rev-parse", "--verify", "HEAD^{commit}"); err == nil {
		return true, nil
	}
	all, err := git.OutputC("rev-list", "--all", "-1")
	if err == nil && strings.TrimSpace(all) == "" {
		return false, nil
	}
	out, err := git.Output("rev-list", "-1", "HEAD")
	if err == nil {
		return true, nil
	}
	out = strings.TrimSpace(out)
	if out != "" {
		return false, errors.New(out)
	}
	return false, err
}

func runDetectedWork(cmd *cobra.Command) (workcmd.Detected, error) {
	work, _, err := cmd.Find([]string{"work"})
	if err != nil {
		return workcmd.Detected{}, err
	}
	cliruntime.Bind(cmd, work)
	return workcmd.RunDetected(work)
}

func repoPickerLabel() string {
	out, err := git.Output("rev-parse", "--show-toplevel")
	if err != nil {
		return "eg"
	}
	line := strings.TrimSpace(out)
	if line == "" || strings.ContainsAny(line, "\r\n") {
		return "eg"
	}
	name := filepath.Base(line)
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "eg"
	}
	return name
}

func pickerDefault(choices []prompt.Choice, ran string) string {
	skip := ""
	if ran != "" {
		skip = "work " + ran
	}
	for _, c := range choices {
		if c.Value != skip {
			return c.Value
		}
	}
	return "quit"
}

func rootChoices(workActions []string) []prompt.Choice {
	var choices []prompt.Choice
	add := func(object string, actions []string) {
		for _, action := range actions {
			choices = append(choices, prompt.Choice{
				Value:       object + " " + action,
				Description: clicatalog.Purpose(object, action),
			})
		}
	}
	add("work", workActions)
	add("repo", repocmd.RelevantActions())
	add("workspace", workspacecmd.RelevantActions())
	add("hook", catalogActions("hook"))
	add("release", catalogActions("release"))
	choices = append(choices,
		prompt.Choice{Value: "help", Description: clicatalog.Purpose("eg", "help")},
		prompt.Choice{Value: "quit", Description: clicatalog.Purpose("eg", "quit")},
	)
	return choices
}

func catalogActions(object string) []string {
	for _, g := range clicatalog.Groups {
		if g.Object != object {
			continue
		}
		out := make([]string, len(g.Commands))
		for i, c := range g.Commands {
			out[i] = c.Action
		}
		return out
	}
	return nil
}

func dispatchRoot(cmd *cobra.Command, object, action string, args []string) error {
	sub, _, err := cmd.Find([]string{object, action})
	if err != nil {
		return cliruntime.NewUsageError(cmd, err)
	}
	if sub == cmd || sub.Name() != action {
		return cliruntime.NewUsageError(cmd, fmt.Errorf("unknown command %q", object+" "+action))
	}
	return cliruntime.RunBound(cmd, sub, args)
}

// Execute runs the eg CLI.
func Execute() {
	defer deprecation.Flush()
	if err := rootCmd.Execute(); err != nil {
		if isUnknownCommand(err) {
			name := unknownCommandName(err)
			fmt.Fprintf(os.Stderr, "Unknown command: eg %s\n", name)
			clicatalog.WriteRootUsage(os.Stderr)
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
		clicatalog.WriteRootUsage(cmd.OutOrStdout())
	})
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	rootCmd.PersistentFlags().BoolVar(&workflows.Skip, "no-workflows", false, "disables available workflows")
	rootCmd.PersistentFlags().BoolVar(&nonInteractive, "non-interactive", false, "disable prompts; fail when required input is missing")
	rootCmd.PersistentFlags().BoolVar(&forceInteractive, "interactive", false, "force prompts on a TTY (overrides CI and --non-interactive)")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		recordDeprecatedSurface(cmd)
		git.Use(git.RealRunner{})
		nested := invocationNested()
		if !skipAuto(cmd) {
			migrate.Auto()
		}
		if err := guardInvocationDepth(); err != nil {
			return err
		}
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
		if shouldAutoConfigure(cmd, nested) {
			if err := selfcmd.AutoConfigure(cmd); err != nil {
				return err
			}
		}
		return nil
	}

	rootCmd.AddCommand(versioncmd.NewCommand())
	rootCmd.AddCommand(completioncmd.NewCommand())

	selfCmd := selfcmd.NewCommand()
	AttachObjectGroup(selfCmd, "self")
	rootCmd.AddCommand(selfCmd)

	repoCmd := repocmd.NewCommand()
	AttachObjectHelp(repoCmd, "repo")
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
	legacyshim.RegisterObjectShims(rootCmd)
	cliruntime.ConfigureCommandTree(rootCmd)
}

func isUnknownCommand(err error) bool {
	return strings.Contains(err.Error(), "unknown command")
}

func recordDeprecatedSurface(cmd *cobra.Command) {
	type rename struct {
		id          string
		replacement string
	}
	renames := map[string]rename{
		"memory profiles":     {id: deprecation.DEP012, replacement: "workspace list all"},
		"workspace create":    {id: deprecation.DEP012, replacement: "workspace new"},
		"workspace status":    {id: deprecation.DEP017, replacement: "workspace list current"},
		"memory status":       {id: deprecation.DEP018, replacement: "self list"},
		"git status":          {id: deprecation.DEP018, replacement: "self list"},
		"repo status":         {id: deprecation.DEP018, replacement: "repo list"},
		"hook status":         {id: deprecation.DEP018, replacement: "hook list"},
		"git":                 {id: deprecation.DEP019, replacement: "self"},
		"git configure":       {id: deprecation.DEP019, replacement: "self configure"},
		"git list":            {id: deprecation.DEP019, replacement: "self list"},
		"git doctor":          {id: deprecation.DEP019, replacement: "self doctor"},
		"memory":              {id: deprecation.DEP019, replacement: "self"},
		"memory list":         {id: deprecation.DEP019, replacement: "self list"},
		"memory workspaces":   {id: deprecation.DEP019, replacement: "workspace list all"},
		"memory repositories": {id: deprecation.DEP019, replacement: "repo list all"},
		"work amend":          {id: deprecation.DEP020, replacement: "work save"},
	}
	for c := cmd; c != nil; c = c.Parent() {
		surface := c.Annotations[deprecation.SurfaceAnnotation]
		if surface == "" {
			continue
		}
		if r, ok := renames[surface]; ok {
			deprecation.RecordRenamed(r.id, surface, r.replacement)
			return
		}
		replacement := "workspace"
		if strings.HasPrefix(surface, "profile") {
			replacement = strings.Replace(surface, "profile", "workspace", 1)
		}
		deprecation.RecordRenamedSurface(surface, replacement, "")
		return
	}
}

func shouldAutoConfigure(cmd *cobra.Command, nested bool) bool {
	return !nested && !skipAuto(cmd)
}

func invocationNested() bool {
	d, err := strconv.Atoi(os.Getenv(invocationDepthEnv))
	if err != nil {
		return false
	}
	return d > 0
}

func skipAuto(cmd *cobra.Command) bool {
	if flagTrue(cmd, "help") || flagTrue(cmd.Root(), "version") {
		return true
	}
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "completion", "version", "help":
			return true
		case "migrate":
			if flagTrue(c, "dry-run") {
				return true
			}
		}
	}
	return false
}

func flagTrue(cmd *cobra.Command, name string) bool {
	if cmd == nil {
		return false
	}
	f := cmd.Flags().Lookup(name)
	if f == nil {
		f = cmd.PersistentFlags().Lookup(name)
	}
	if f == nil {
		return false
	}
	v, err := strconv.ParseBool(f.Value.String())
	return err == nil && v
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
