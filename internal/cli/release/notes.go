package release

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

var notesID = cmdid.ID{Command: "release", Action: "notes"}

func releaseNotesSpec(layout, fromRef, toRef *string) argspec.Spec {
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInputWithComplete("layout", 0, false, "Release notes layout", layout, func() string { return "simple" }, sources.ReleaseNotesLayouts, false),
		argspec.PositionalInputWithComplete("from-ref", 1, false, "From ref", fromRef, nil, sources.Refs, true),
		argspec.PositionalInputWithComplete("to-ref", 2, false, "To ref", toRef, func() string { return "HEAD" }, sources.Refs, true),
	}}
}

func newNotesCommand() *cobra.Command {
	var layout, fromRef, toRef string
	spec := releaseNotesSpec(&layout, &fromRef, &toRef)
	c := &cobra.Command{
		Use:   "notes [<layout>] [<from-ref>] [<to-ref>]",
		Short: "Prints a release log between two refs",
		Long:  "Prints release notes between two refs using a simple or smart (GitHub) layout.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliruntime.RunWithWorkflows(cmd, notesID, func() error {
				return notesRun(cmd, args)
			})
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.AttachArgs(c, spec)
	return c
}

func notesRun(cmd *cobra.Command, args []string) error {
	var layout, fromRef, toRef string
	if err := argspec.ResolveCmd(cmd, args, releaseNotesSpec(&layout, &fromRef, &toRef)); err != nil {
		return err
	}
	resolved := []string{}
	if layout != "" {
		resolved = append(resolved, layout)
	}
	if fromRef != "" {
		resolved = append(resolved, fromRef)
	}
	if toRef != "" {
		resolved = append(resolved, toRef)
	}
	out, err := formatReleaseNotes(resolved)
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, out)
	return nil
}

func formatReleaseNotes(args []string) (string, error) {
	layout := "simple"
	if len(args) > 0 && args[0] != "" {
		layout = args[0]
	}
	first := state.LastTag()
	if len(args) > 1 && args[1] != "" {
		first = args[1]
	}
	second := "HEAD"
	if len(args) > 2 && args[2] != "" {
		second = args[2]
	}
	diapason := first + "..." + second
	if first == "all-commits" || first == "" {
		diapason = second
	}
	switch layout {
	case "simple":
		return simpleReleaseNotes(diapason), nil
	case "smart":
		url := git.OutputOK("remote", "get-url", "origin")
		if isGitHubRemote(url) {
			return githubReleaseNotes(diapason, url), nil
		}
		return simpleReleaseNotes(diapason), nil
	default:
		cliruntime.ExitWorkflowError(fmt.Sprintf("A layout can be 'simple' or 'smart'! '%s' layout is not supported.", layout))
		return "", nil
	}
}

func isGitHubRemote(url string) bool {
	return strings.Contains(url, "github.com")
}

func simpleReleaseNotes(diapason string) string {
	var b strings.Builder
	b.WriteString("Release notes\n")
	for _, hash := range commitHashes(diapason) {
		subject := git.OutputOK("show", "-s", "--pretty=%s", hash)
		b.WriteString("- " + subject + "\n")
	}
	return b.String()
}

func githubReleaseNotes(diapason, remoteURL string) string {
	repo := githubRepository(remoteURL)
	var b strings.Builder
	b.WriteString("<h3>Release notes</h3>\n")
	for _, hash := range commitHashes(diapason) {
		body := git.OutputOK("show", "-s", "--pretty=%B", hash)
		issues := ""
		if idx := strings.Index(body, "#"); idx >= 0 {
			issues = " [" + strings.TrimSpace(body[idx:]) + "]"
		}
		subject := git.OutputOK("show", "-s", "--pretty=%s", hash)
		fmt.Fprintf(&b, "<li> <a href=\"https://github.com/%s/commit/%s\">%s</a>%s</li>\n", repo, hash, subject, issues)
	}
	return b.String()
}

func githubRepository(url string) string {
	re := regexp.MustCompile(`github\.com[:/]([^/]+/[^/]+?)(?:\.git)?$`)
	if m := re.FindStringSubmatch(url); len(m) > 1 {
		return strings.TrimSuffix(m[1], ".git")
	}
	repo := url
	if i := strings.Index(repo, "github.com/"); i >= 0 {
		repo = repo[i+len("github.com/"):]
	} else if i := strings.Index(repo, "github.com:"); i >= 0 {
		repo = repo[i+len("github.com:"):]
	}
	return strings.TrimSuffix(repo, ".git")
}

func commitHashes(diapason string) []string {
	out := git.OutputOK("log", diapason, "--format=%H", "--reverse")
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}
