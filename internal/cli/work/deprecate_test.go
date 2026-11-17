package work

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/cli/catalog"
	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/runtime"
	"github.com/spf13/cobra"
)

func TestTrackM13Checkout(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.addRemote("origin", "main", "feat")
	r.git("checkout", "-q", "main")
	r.git("branch", "-D", "feat")
	r.git("update-ref", "-d", "refs/remotes/origin/feat")
	r.enter()
	c := newTrackCommand()
	if !c.Hidden {
		t.Fatal("work track should be hidden")
	}
	if c.Annotations[deprecation.SurfaceAnnotation] != "work track" {
		t.Fatalf("annotation %q", c.Annotations[deprecation.SurfaceAnnotation])
	}
	if err := runTrackCmd(prompt.NewNonInteractive(), "origin/feat"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/feat" {
		t.Fatalf("head %s", r.headRef())
	}
	if r.upstream("feat") != "origin/feat" {
		t.Fatalf("upstream %s", r.upstream("feat"))
	}
}

func TestTrackM14AbsentFromHelp(t *testing.T) {
	for _, g := range catalog.Groups {
		if g.Object != "work" {
			continue
		}
		for _, c := range g.Commands {
			if c.Action == "track" {
				t.Fatal("catalog lists track")
			}
		}
	}
	var workHelp, rootHelp bytes.Buffer
	catalog.WriteObjectUsage(&workHelp, "work")
	catalog.WriteRootUsage(&rootHelp)
	if actionListed(workHelp.String(), "track") || actionListed(rootHelp.String(), "track") {
		t.Fatalf("help lists track\nwork:\n%s\nroot:\n%s", workHelp.String(), rootHelp.String())
	}
	if !actionListed(workHelp.String(), "switch") || !actionListed(rootHelp.String(), "switch") {
		t.Fatal("help missing switch")
	}
}

func TestTrackM17Hooks(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.addRemote("origin", "main", "feat")
	r.git("checkout", "-q", "main")
	r.git("branch", "-D", "feat")
	r.git("update-ref", "-d", "refs/remotes/origin/feat")
	marker := filepath.Join(t.TempDir(), "hook")
	writeNamedHook(t, r.dir, "work-track-ahead", marker, "track")
	r.enter()
	if err := runTrackCmd(prompt.NewNonInteractive(), "origin/feat"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readMarker(t, marker), "track") {
		t.Fatalf("track hook did not run: %q", readMarker(t, marker))
	}
	if err := os.WriteFile(marker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runSwitchCmd(prompt.NewNonInteractive(), "main"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(readMarker(t, marker), "track") {
		t.Fatalf("switch ran work-track-ahead: %q", readMarker(t, marker))
	}
}

func TestAcceptM17RunsTrackAheadHook(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "ran")
	dir := filepath.Join(root, ".config", "elegant-git", "hooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf("#!/bin/bash\necho accept >> %q\n", marker)
	if err := os.WriteFile(filepath.Join(dir, "work-track-ahead"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	m := setupAcceptTest(t)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["rev-parse --show-toplevel"] = root
	m.FailOn["rev-parse --verify --quiet --abbrev-ref --branches=refs/heads origin/feat"] = fmt.Errorf("unknown")
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "main"
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/feat"

	var got string
	spec := acceptSpec(&got)
	cmd := &cobra.Command{}
	ctx := prompt.WithPrompter(context.Background(), prompt.NewNonInteractive())
	ctx = runtime.WithRepoLayout(ctx, runtime.RepoLayout{RepoRoot: root})
	cmd.SetContext(ctx)
	if err := acceptLogic(cmd, []string{"origin/feat"}, spec); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(marker)
	if err != nil || !strings.Contains(string(b), "accept") {
		t.Fatalf("marker %q err %v", b, err)
	}
}

func runTrackCmd(p prompt.Prompter, args ...string) error {
	cmd := newTrackCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), p))
	return cmd.RunE(cmd, args)
}

func writeNamedHook(t *testing.T, repo, name, marker, word string) {
	t.Helper()
	dir := filepath.Join(repo, ".config", "elegant-git", "hooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf("#!/bin/bash\necho %s >> %q\n", word, marker)
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	ignoreRepoConfig(t, repo)
}

func actionListed(text, action string) bool {
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == action {
			return true
		}
	}
	return false
}
