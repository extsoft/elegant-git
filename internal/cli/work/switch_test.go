package work

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

func TestSwitchM1LocalNoFetch(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.git("checkout", "-q", "main")
	r.addRemote("origin", "main")
	tip := r.git("rev-parse", "HEAD")
	gitDir(t, r.remoteURL("origin"), "update-ref", "refs/heads/onlyremote", tip)
	r.enter()
	if err := runSwitch(prompt.NewNonInteractive(), "feat"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/feat" {
		t.Fatalf("head %s", r.headRef())
	}
	if r.gitAllow("rev-parse", "--verify", "--quiet", "refs/remotes/origin/onlyremote") == nil {
		t.Fatal("fetched")
	}
}

func TestSwitchM2RemoteTracking(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.addRemote("origin", "main", "feat")
	r.git("checkout", "-q", "main")
	r.git("branch", "-D", "feat")
	r.git("update-ref", "-d", "refs/remotes/origin/feat")
	r.enter()
	if err := runSwitch(prompt.NewNonInteractive(), "origin/feat"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/feat" {
		t.Fatalf("head %s", r.headRef())
	}
	if r.upstream("feat") != "origin/feat" {
		t.Fatalf("upstream %s", r.upstream("feat"))
	}
}

func TestSwitchM3UniqueRemoteName(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.addRemote("origin", "main", "feat")
	r.git("checkout", "-q", "main")
	r.git("branch", "-D", "feat")
	r.enter()
	if err := runSwitch(prompt.NewNonInteractive(), "feat"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/feat" {
		t.Fatalf("head %s", r.headRef())
	}
	if r.upstream("feat") != "origin/feat" {
		t.Fatalf("upstream %s", r.upstream("feat"))
	}
}

func TestSwitchM4AmbiguousRemotes(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.git("checkout", "-q", "main")
	r.addRemote("origin", "main", "feat")
	r.pushRemote("upstream", "main", "feat")
	r.git("branch", "-D", "feat")
	r.enter()
	buf := captureText(t)
	err := runSwitch(prompt.NewNonInteractive(), "feat")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "origin/feat") || !strings.Contains(buf.String(), "upstream/feat") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.headRef() != "refs/heads/main" {
		t.Fatalf("head %s", r.headRef())
	}
}

func TestSwitchM5Substring(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "feature")
	r.git("checkout", "-q", "main")
	r.enter()
	if err := runSwitch(prompt.NewNonInteractive(), "fea"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/feature" {
		t.Fatalf("head %s", r.headRef())
	}
	r.git("branch", "feat")
	buf := captureText(t)
	err := runSwitch(prompt.NewNonInteractive(), "fea")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "feat") || !strings.Contains(buf.String(), "feature") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.headRef() != "refs/heads/feature" {
		t.Fatalf("head %s", r.headRef())
	}
	buf.Reset()
	err = runSwitch(prompt.NewNonInteractive(), "zzz")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "'zzz' is not a branch, tag, or commit") {
		t.Fatalf("out=%q", buf.String())
	}
}

func TestSwitchM6Behind(t *testing.T) {
	r := behindFeat(t)
	r.enter()
	want := r.rev("refs/remotes/origin/feat")
	if err := runSwitch(prompt.NewNonInteractive(), "origin/feat"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/feat" || r.rev("refs/heads/feat") != want {
		t.Fatalf("head %s tip %s want %s", r.headRef(), r.rev("refs/heads/feat"), want)
	}

	r = behindFeat(t)
	r.enter()
	want = r.rev("refs/remotes/origin/feat")
	p := &swPrompter{picks: []string{"fast-forward"}}
	if err := runSwitch(p, "origin/feat"); err != nil {
		t.Fatal(err)
	}
	if len(p.pickDefs) != 1 || p.pickDefs[0] != "fast-forward" {
		t.Fatalf("defaults=%v", p.pickDefs)
	}
	if r.rev("refs/heads/feat") != want {
		t.Fatalf("tip %s want %s", r.rev("refs/heads/feat"), want)
	}
}

func TestSwitchM7Diverged(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.commit("feat", "g", "feat\n")
	r.addRemote("origin", "main", "feat")
	r.commit("local", "h", "local\n")
	tip := r.rev("HEAD")
	r.git("checkout", "-q", "main")
	r.commit("main", "f", "moved\n")
	r.git("push", "-q", "--force", "origin", "main:feat")
	r.git("checkout", "-q", "main")
	r.enter()
	buf := captureText(t)
	if err := runSwitch(prompt.NewNonInteractive(), "origin/feat"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/feat" {
		t.Fatalf("head %s", r.headRef())
	}
	if r.rev("refs/heads/feat") != tip {
		t.Fatalf("tip moved %s want %s", r.rev("refs/heads/feat"), tip)
	}
	if !strings.Contains(buf.String(), "has diverged from 'origin/feat'") {
		t.Fatalf("out=%q", buf.String())
	}
}

func TestSwitchM8Dirty(t *testing.T) {
	t.Run("add", func(t *testing.T) {
		r := dirtyPair(t)
		r.enter()
		if err := runSwitch(&swPrompter{picks: []string{"add"}}, "feat"); err != nil {
			t.Fatal(err)
		}
		if r.headRef() != "refs/heads/feat" || r.read("f") != "dirty\n" {
			t.Fatalf("head %s file %q", r.headRef(), r.read("f"))
		}
	})
	t.Run("reset", func(t *testing.T) {
		r := dirtyPair(t)
		r.enter()
		if err := runSwitch(&swPrompter{picks: []string{"reset"}}, "feat"); err != nil {
			t.Fatal(err)
		}
		if r.headRef() != "refs/heads/feat" || r.read("f") != "base\n" {
			t.Fatalf("head %s file %q", r.headRef(), r.read("f"))
		}
	})
	t.Run("cancel", func(t *testing.T) {
		r := dirtyPair(t)
		r.enter()
		err := runSwitch(&swPrompter{picks: []string{"cancel"}}, "feat")
		if err == nil || !strings.Contains(err.Error(), "cancelled") {
			t.Fatalf("err=%v", err)
		}
		if r.headRef() != "refs/heads/main" || r.read("f") != "dirty\n" || r.stashList() != "" {
			t.Fatalf("head %s file %q stash %q", r.headRef(), r.read("f"), r.stashList())
		}
	})
	t.Run("non-interactive", func(t *testing.T) {
		r := dirtyPair(t)
		r.enter()
		if err := runSwitch(prompt.NewNonInteractive(), "feat"); err != nil {
			t.Fatal(err)
		}
		if r.headRef() != "refs/heads/feat" || r.read("f") != "dirty\n" {
			t.Fatalf("head %s file %q", r.headRef(), r.read("f"))
		}
	})
}

func TestSwitchM16NoRef(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.git("checkout", "-q", "main")
	r.git("tag", "v1")
	r.addRemote("origin", "main")
	r.enter()
	p := &swPrompter{picks: []string{"feat"}}
	if err := runSwitch(p, ""); err != nil {
		t.Fatal(err)
	}
	if len(p.choices) != 1 {
		t.Fatalf("picks=%d", len(p.choices))
	}
	got := choiceValues(p.choices[0])
	for _, want := range []string{"feat", "origin/main", "v1"} {
		if !containsName(got, want) {
			t.Fatalf("choices %v lack %s", got, want)
		}
	}
	if r.headRef() != "refs/heads/feat" {
		t.Fatalf("head %s", r.headRef())
	}
	err := runSwitch(prompt.NewNonInteractive())
	var missing *argspec.ErrMissingRequired
	if !errors.As(err, &missing) || len(missing.Names) != 1 || missing.Names[0] != "ref" {
		t.Fatalf("err=%v", err)
	}
}

func TestSwitchM18WorktreeRestores(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.git("checkout", "-q", "main")
	wt := t.TempDir()
	r.git("worktree", "add", "-q", wt, "feat")
	r.write("f", "dirty\n")
	r.enter()
	err, stderr := withStderr(t, func() error {
		return runSwitch(prompt.NewNonInteractive(), "feat")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var es cliruntime.ExitStatus
	if errors.As(err, &es) {
		t.Fatalf("exit %d stderr %s", es.Code, stderr)
	}
	if !strings.Contains(stderr, "feat") {
		t.Fatalf("stderr %q err %v", stderr, err)
	}
	if r.headRef() != "refs/heads/main" || r.read("f") != "dirty\n" || r.stashList() != "" {
		t.Fatalf("head %s file %q stash %q", r.headRef(), r.read("f"), r.stashList())
	}
}

func TestSwitchM19CurrentAndLocalArg(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.enter()
	buf := captureText(t)
	if err := runSwitch(prompt.NewNonInteractive(), "feat"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Already on 'feat'") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.headRef() != "refs/heads/feat" {
		t.Fatalf("head %s", r.headRef())
	}
	err := runSwitch(prompt.NewNonInteractive(), "feat", "other")
	if !cliruntime.IsUsageError(err) {
		t.Fatalf("err=%v", err)
	}
	if r.headRef() != "refs/heads/feat" {
		t.Fatalf("head %s", r.headRef())
	}
}

func TestSwitchM20InProgress(t *testing.T) {
	tests := []struct {
		name  string
		op    string
		abort string
		setup func(*swRepo)
	}{
		{"merge", "merge", "git merge --abort", setupConflictMerge},
		{"cherry-pick", "cherry-pick", "git cherry-pick --abort", setupConflictCherryPick},
		{"revert", "revert", "git revert --abort", setupConflictRevert},
		{"rebase", "rebase", "git rebase --abort", setupConflictRebase},
		{"bisect", "bisect", "git bisect reset", setupBisect},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := newSwRepo(t)
			r.git("branch", "other")
			tc.setup(r)
			r.enter()
			before := r.rev("HEAD")
			buf := captureText(t)
			err := runSwitch(prompt.NewNonInteractive(), "other")
			assertWorkflow(t, err)
			out := buf.String()
			if !strings.Contains(out, tc.op) || !strings.Contains(out, tc.abort) {
				t.Fatalf("out=%q", out)
			}
			if r.rev("HEAD") != before || r.stashList() != "" {
				t.Fatalf("head moved or stashed: %s stash %q", r.rev("HEAD"), r.stashList())
			}
		})
	}
}

func TestSwitchM21CompletionNoFetch(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["for-each-ref --format=%(refname:short)\t%(upstream:short) refs/heads"] = "feat\torigin/feat\nmain\torigin/main"
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/main\norigin/HEAD"
	m.Outputs["for-each-ref --format=%(refname:short) refs/tags"] = "v1"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	cmd := newSwitchCommand()
	if cmd.ValidArgsFunction == nil {
		t.Fatal("expected ValidArgsFunction")
	}
	out, dir := cmd.ValidArgsFunction(cmd, nil, "")
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("dir=%v", dir)
	}
	got := strings.Join(out, "\n")
	for _, want := range []string{"feat", "main", "origin/main", "v1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("completion %q lacks %s", got, want)
		}
	}
	for _, c := range m.Calls {
		if len(c.Args) > 0 && c.Args[0] == "fetch" {
			t.Fatalf("fetched %v", m.Calls)
		}
	}
}

func TestSwitchM23LeadingDash(t *testing.T) {
	r := newSwRepo(t)
	r.enter()
	for _, ref := range []string{"-x", "--help"} {
		err := runSwitch(prompt.NewNonInteractive(), ref)
		if !cliruntime.IsUsageError(err) {
			t.Fatalf("ref %s err=%v", ref, err)
		}
		if r.headRef() != "refs/heads/main" {
			t.Fatalf("head %s", r.headRef())
		}
	}
	buf := captureText(t)
	for _, ref := range []string{"main..main", "f", "HEAD:f"} {
		buf.Reset()
		err := runSwitch(prompt.NewNonInteractive(), ref)
		assertWorkflow(t, err)
		if !strings.Contains(buf.String(), "'"+ref+"' is not a branch, tag, or commit") || r.headRef() != "refs/heads/main" {
			t.Fatalf("ref %s out=%q head %s", ref, buf.String(), r.headRef())
		}
	}
}

func TestSwitchM9Tag(t *testing.T) {
	r := newSwRepo(t)
	r.git("tag", "v1.2")
	want := r.rev("refs/tags/v1.2^{commit}")
	r.enter()
	if err := runSwitch(prompt.NewNonInteractive(), "v1.2"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/tag-v1.2" || r.rev("refs/heads/tag-v1.2") != want {
		t.Fatalf("head %s tip %s want %s", r.headRef(), r.rev("HEAD"), want)
	}
	if r.sourceOf("tag-v1.2") != "v1.2" {
		t.Fatalf("source %q", r.sourceOf("tag-v1.2"))
	}
	if r.gitAllow("rev-parse", "--abbrev-ref", "tag-v1.2@{upstream}") == nil {
		t.Fatal("tracking set")
	}
}

func TestSwitchTagAsksForName(t *testing.T) {
	r := newSwRepo(t)
	r.git("tag", "v1.2")
	want := r.rev("refs/tags/v1.2^{commit}")
	r.enter()
	p := &swPrompter{lines: []string{"mine"}}
	if err := runSwitch(p, "v1.2"); err != nil {
		t.Fatal(err)
	}
	if len(p.stringDefs) == 0 || p.stringDefs[0] != "tag-v1.2" {
		t.Fatalf("defs %v", p.stringDefs)
	}
	if r.headRef() != "refs/heads/mine" || r.rev("refs/heads/mine") != want || r.sourceOf("mine") != "v1.2" {
		t.Fatalf("head %s source %q", r.headRef(), r.sourceOf("mine"))
	}
}

func TestSwitchM10Revision(t *testing.T) {
	r := newSwRepo(t)
	r.commit("next", "g", "next\n")
	full := r.rev("HEAD~1")
	short := r.git("rev-parse", "--short", full)
	r.enter()
	if err := runSwitch(prompt.NewNonInteractive(), full); err != nil {
		t.Fatal(err)
	}
	name := "rev-" + short
	if r.headRef() != "refs/heads/"+name || r.rev("refs/heads/"+name) != full {
		t.Fatalf("head %s tip %s", r.headRef(), r.rev("HEAD"))
	}
	if r.gitAllow("check-ref-format", "--branch", name) != nil {
		t.Fatalf("invalid branch name %s", name)
	}
	if r.sourceOf(name) != full {
		t.Fatalf("source %q", r.sourceOf(name))
	}

	r = newSwRepo(t)
	r.commit("next", "g", "next\n")
	full = r.rev("HEAD~1")
	short = r.git("rev-parse", "--short", full)
	r.enter()
	if err := runSwitch(prompt.NewNonInteractive(), "HEAD~1"); err != nil {
		t.Fatal(err)
	}
	name = "rev-" + short
	if r.headRef() != "refs/heads/"+name || r.rev("HEAD") != full || r.sourceOf(name) != "HEAD~1" {
		t.Fatalf("head %s tip %s source %q", r.headRef(), r.rev("HEAD"), r.sourceOf(name))
	}
}

func TestSwitchM11TagNameTaken(t *testing.T) {
	t.Run("same commit", func(t *testing.T) {
		r := newSwRepo(t)
		r.git("tag", "v1.2")
		r.git("branch", "tag-v1.2", "v1.2")
		tip := r.rev("refs/heads/tag-v1.2")
		r.enter()
		buf := captureText(t)
		err := runSwitch(prompt.NewNonInteractive(), "v1.2")
		assertWorkflow(t, err)
		if !strings.Contains(buf.String(), "branch 'tag-v1.2' already exists") || r.headRef() != "refs/heads/main" {
			t.Fatalf("out=%q head %s", buf.String(), r.headRef())
		}
		if r.rev("refs/heads/tag-v1.2") != tip {
			t.Fatal("moved existing branch")
		}
	})
	t.Run("elsewhere", func(t *testing.T) {
		r := newSwRepo(t)
		r.commit("next", "g", "next\n")
		r.git("tag", "v1.2", "HEAD~1")
		r.git("branch", "tag-v1.2", "HEAD")
		tip := r.rev("refs/heads/tag-v1.2")
		r.enter()
		buf := captureText(t)
		err := runSwitch(prompt.NewNonInteractive(), "v1.2")
		assertWorkflow(t, err)
		if !strings.Contains(buf.String(), "branch 'tag-v1.2' already exists") || r.headRef() != "refs/heads/main" {
			t.Fatalf("out=%q head %s", buf.String(), r.headRef())
		}
		if r.rev("refs/heads/tag-v1.2") != tip {
			t.Fatal("moved existing branch")
		}
		if r.gitAllow("rev-parse", "--verify", "--quiet", "refs/heads/tag-v1.2-2") == nil {
			t.Fatal("suffixed")
		}
	})
}

func TestSwitchM12LocalBranchArg(t *testing.T) {
	r := newSwRepo(t)
	r.git("tag", "v1.2")
	r.git("branch", "taken")
	want := r.rev("refs/tags/v1.2^{commit}")
	r.enter()
	buf := captureText(t)
	err := runSwitch(prompt.NewNonInteractive(), "v1.2", "taken")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "branch 'taken' already exists") || r.headRef() != "refs/heads/main" {
		t.Fatalf("out=%q head %s", buf.String(), r.headRef())
	}
	if r.gitAllow("rev-parse", "--verify", "--quiet", "refs/heads/tag-v1.2") == nil {
		t.Fatal("auto name created")
	}
	if err := runSwitch(prompt.NewNonInteractive(), "v1.2", "mine"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/mine" || r.rev("refs/heads/mine") != want || r.sourceOf("mine") != "v1.2" {
		t.Fatalf("head %s source %q", r.headRef(), r.sourceOf("mine"))
	}
}

func TestSwitchM22OtherRevision(t *testing.T) {
	r := newSwRepo(t)
	r.commit("next", "g", "next\n")
	full := r.rev("HEAD~1")
	short := r.git("rev-parse", "--short", full)
	r.enter()
	p := &swPrompter{picks: []string{"other revision"}, lines: []string{"HEAD~1"}}
	if err := runSwitch(p, ""); err != nil {
		t.Fatal(err)
	}
	if len(p.choices) != 1 || p.choices[0][len(p.choices[0])-1].Value != "other revision" {
		t.Fatalf("choices %v", choiceValues(p.choices[0]))
	}
	name := "rev-" + short
	if r.headRef() != "refs/heads/"+name || r.rev("HEAD") != full || r.sourceOf(name) != "HEAD~1" {
		t.Fatalf("head %s tip %s source %q", r.headRef(), r.rev("HEAD"), r.sourceOf(name))
	}

	r.git("checkout", "-q", "main")
	buf := captureText(t)
	p = &swPrompter{picks: []string{"other revision"}, lines: []string{"nope"}}
	err := runSwitch(p, "")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "'nope' is not a branch, tag, or commit") || r.headRef() != "refs/heads/main" {
		t.Fatalf("out=%q head %s", buf.String(), r.headRef())
	}
}

func TestSwitchM24AnnotatedTag(t *testing.T) {
	r := newSwRepo(t)
	r.git("tag", "-a", "v2", "-m", "note")
	tagObj := r.rev("refs/tags/v2")
	commit := r.rev("refs/tags/v2^{commit}")
	if tagObj == commit {
		t.Fatal("expected annotated tag")
	}
	r.enter()
	if err := runSwitch(prompt.NewNonInteractive(), "v2"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/tag-v2" || r.rev("refs/heads/tag-v2") != commit || r.rev("refs/heads/tag-v2") == tagObj {
		t.Fatalf("head %s tip %s commit %s tag %s", r.headRef(), r.rev("HEAD"), commit, tagObj)
	}
}

func TestSwitchM27Suffix(t *testing.T) {
	t.Run("case", func(t *testing.T) {
		r := newSwRepo(t)
		r.git("config", "core.ignorecase", "true")
		r.git("branch", "Tag-v1")
		r.git("tag", "v1")
		r.enter()
		if r.gitAllow("show-ref", "--verify", "--quiet", "refs/heads/tag-v1") != nil {
			t.Skip("case-insensitive branch lookup is unavailable")
		}
		buf := captureText(t)
		err := runSwitch(prompt.NewNonInteractive(), "v1")
		assertWorkflow(t, err)
		if !strings.Contains(buf.String(), "branch 'tag-v1' already exists") || r.headRef() != "refs/heads/main" {
			t.Fatalf("out=%q head %s", buf.String(), r.headRef())
		}
	})
	t.Run("namespace", func(t *testing.T) {
		r := newSwRepo(t)
		r.git("branch", "tag-v1/x")
		r.git("tag", "v1")
		r.enter()
		buf := captureText(t)
		err := runSwitch(prompt.NewNonInteractive(), "v1")
		assertWorkflow(t, err)
		if !strings.Contains(buf.String(), "branch 'tag-v1' conflicts with existing branch 'tag-v1/x'") || r.headRef() != "refs/heads/main" {
			t.Fatalf("out=%q head %s", buf.String(), r.headRef())
		}
	})
}

func TestSwitchExactTagBeatsSubstring(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "v1-fix")
	r.addRemote("origin", "main", "v1-fix")
	r.git("checkout", "-q", "main")
	r.git("branch", "-D", "v1-fix")
	r.git("tag", "v1")
	want := r.rev("refs/tags/v1^{commit}")
	r.enter()
	if err := runSwitch(prompt.NewNonInteractive(), "v1"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/tag-v1" || r.rev("refs/heads/tag-v1") != want {
		t.Fatalf("head %s tip %s", r.headRef(), r.rev("HEAD"))
	}
	if r.gitAllow("rev-parse", "--verify", "--quiet", "refs/heads/v1-fix") == nil {
		t.Fatal("substring branch created")
	}
}

func TestSwitchM25Kinds(t *testing.T) {
	r := newSwRepo(t)
	r.git("branch", "x")
	r.git("tag", "x")
	r.addRemote("origin", "main")
	r.git("push", "-q", "origin", "refs/heads/x:refs/heads/x")
	r.enter()
	buf := captureText(t)
	if err := runSwitch(prompt.NewNonInteractive(), "x"); err != nil {
		t.Fatalf("%v out=%q", err, buf.String())
	}
	if r.headRef() != "refs/heads/x" {
		t.Fatalf("head %s", r.headRef())
	}
	if !strings.Contains(buf.String(), "tags/x") {
		t.Fatalf("out=%q", buf.String())
	}
	r.git("checkout", "-q", "main")
	buf.Reset()
	if err := runSwitch(prompt.NewNonInteractive(), "tags/x"); err != nil {
		t.Fatalf("%v out=%q", err, buf.String())
	}
	if r.headRef() != "refs/heads/tag-x" || r.rev("refs/heads/tag-x") != r.rev("refs/tags/x^{commit}") {
		t.Fatalf("head %s", r.headRef())
	}
	if r.sourceOf("tag-x") != "tags/x" {
		t.Fatalf("source %q", r.sourceOf("tag-x"))
	}
	r.git("checkout", "-q", "main")
	r.git("branch", "-D", "tag-x")
	if err := runSwitch(prompt.NewNonInteractive(), "refs/tags/x"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/tag-x" || r.sourceOf("tag-x") != "refs/tags/x" {
		t.Fatalf("head %s source %q", r.headRef(), r.sourceOf("tag-x"))
	}
	r.git("checkout", "-q", "main")
	if err := runSwitch(prompt.NewNonInteractive(), "refs/heads/x"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/x" {
		t.Fatalf("head %s", r.headRef())
	}
	r.git("checkout", "-q", "main")
	r.git("branch", "-D", "x")
	if err := runSwitch(prompt.NewNonInteractive(), "remotes/origin/x"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/x" || r.upstream("x") != "origin/x" {
		t.Fatalf("head %s upstream %s", r.headRef(), r.upstream("x"))
	}
}

func TestSwitchM26UnbornAndBare(t *testing.T) {
	t.Run("clean", func(t *testing.T) {
		src := newSwRepo(t)
		src.git("checkout", "-q", "-b", "feat")
		src.commit("feat", "g", "feat\n")
		bare := t.TempDir()
		src.git("clone", "--bare", "-q", src.dir, bare)
		r := newUnborn(t)
		r.git("remote", "add", "origin", bare)
		r.git("fetch", "-q", "origin")
		r.enter()
		if err := runSwitch(prompt.NewNonInteractive(), "origin/feat"); err != nil {
			t.Fatal(err)
		}
		if r.headRef() != "refs/heads/feat" {
			t.Fatalf("head %s", r.headRef())
		}
	})
	t.Run("dirty", func(t *testing.T) {
		src := newSwRepo(t)
		src.git("checkout", "-q", "-b", "feat")
		bare := t.TempDir()
		src.git("clone", "--bare", "-q", src.dir, bare)
		r := newUnborn(t)
		r.git("remote", "add", "origin", bare)
		r.git("fetch", "-q", "origin")
		r.write("dirty", "x\n")
		r.enter()
		buf := captureText(t)
		err := runSwitch(prompt.NewNonInteractive(), "origin/feat")
		assertWorkflow(t, err)
		if !strings.Contains(buf.String(), "no commits") {
			t.Fatalf("out=%q", buf.String())
		}
		if r.read("dirty") != "x\n" || r.gitAllow("rev-parse", "--verify", "--quiet", "refs/heads/feat") == nil {
			t.Fatal("branch created or file lost")
		}
	})
	t.Run("bare", func(t *testing.T) {
		bare := t.TempDir()
		cmd := exec.Command(git.Executable(), "init", "--bare", "-q", "-b", "main", bare)
		withoutRepoEnv(cmd)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("bare: %v\n%s", err, out)
		}
		clearRepoEnv(t)
		t.Chdir(bare)
		git.Use(git.RealRunner{})
		t.Cleanup(func() { git.Use(git.RealRunner{}) })
		buf := captureText(t)
		err := runSwitch(prompt.NewNonInteractive(), "main")
		assertWorkflow(t, err)
		if !strings.Contains(buf.String(), "no working tree") {
			t.Fatalf("out=%q", buf.String())
		}
	})
}

func TestSwitchM28Hooks(t *testing.T) {
	t.Run("after exit ignored", func(t *testing.T) {
		r := newSwRepo(t)
		r.git("checkout", "-q", "-b", "feat")
		r.git("checkout", "-q", "main")
		marker := filepath.Join(t.TempDir(), "hook")
		writeSwitchHook(t, r.dir, "after", marker, "after")
		r.enter()
		if err := runSwitchCmd(prompt.NewNonInteractive(), "feat"); err != nil {
			t.Fatal(err)
		}
		if r.headRef() != "refs/heads/feat" {
			t.Fatalf("head %s", r.headRef())
		}
		if !strings.Contains(readMarker(t, marker), "after") {
			t.Fatalf("marker %q", readMarker(t, marker))
		}
	})
	t.Run("failed switch still runs after", func(t *testing.T) {
		r := newSwRepo(t)
		marker := filepath.Join(t.TempDir(), "hook")
		writeSwitchHook(t, r.dir, "ahead", marker, "ahead")
		writeSwitchHook(t, r.dir, "after", marker, "after")
		r.enter()
		buf := captureText(t)
		err := runSwitchCmd(prompt.NewNonInteractive(), "nope")
		assertWorkflow(t, err)
		got := readMarker(t, marker)
		if !strings.Contains(got, "ahead") || !strings.Contains(got, "after") {
			t.Fatalf("marker %q out %q", got, buf.String())
		}
		if r.headRef() != "refs/heads/main" {
			t.Fatalf("head %s", r.headRef())
		}
	})
}

func behindFeat(t *testing.T) *swRepo {
	t.Helper()
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.commit("feat", "g", "feat\n")
	r.addRemote("origin", "main", "feat")
	r.commit("ahead", "g", "ahead\n")
	r.git("push", "-q", "origin", "feat")
	r.git("reset", "--hard", "HEAD~1")
	r.git("checkout", "-q", "main")
	return r
}

func dirtyPair(t *testing.T) *swRepo {
	t.Helper()
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.git("checkout", "-q", "main")
	r.write("f", "dirty\n")
	return r
}

func setupConflictMerge(r *swRepo) {
	r.write("f", "main\n")
	r.git("commit", "-q", "-am", "main-side")
	r.git("checkout", "-q", "-b", "side", "HEAD~1")
	r.write("f", "side\n")
	r.git("commit", "-q", "-am", "side")
	r.git("checkout", "-q", "main")
	r.mustConflict("merge", "side")
}

func setupConflictCherryPick(r *swRepo) {
	r.git("checkout", "-q", "-b", "side")
	r.write("f", "side\n")
	r.git("commit", "-q", "-am", "side")
	r.git("checkout", "-q", "main")
	r.write("f", "main\n")
	r.git("commit", "-q", "-am", "main")
	r.mustConflict("cherry-pick", "side")
}

func setupConflictRevert(r *swRepo) {
	r.write("f", "mid\n")
	r.git("commit", "-q", "-am", "mid")
	r.write("f", "tip\n")
	r.git("commit", "-q", "-am", "tip")
	r.mustConflict("revert", "--no-edit", "HEAD~1")
}

func setupConflictRebase(r *swRepo) {
	r.git("checkout", "-q", "-b", "feat")
	r.write("f", "feat\n")
	r.git("commit", "-q", "-am", "feat")
	r.git("checkout", "-q", "main")
	r.write("f", "main\n")
	r.git("commit", "-q", "-am", "main")
	r.git("checkout", "-q", "feat")
	r.mustConflict("rebase", "main")
}

func setupBisect(r *swRepo) {
	r.commit("b", "g", "b\n")
	r.commit("c", "h", "c\n")
	r.git("bisect", "start", "HEAD", "HEAD~2")
}

type swRepo struct {
	t   *testing.T
	dir string
}

func newSwRepo(t *testing.T) *swRepo {
	t.Helper()
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
	dir := t.TempDir()
	r := &swRepo{t: t, dir: dir}
	r.git("init", "-q", "-b", "main")
	r.commit("base", "f", "base\n")
	return r
}

func newUnborn(t *testing.T) *swRepo {
	t.Helper()
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
	dir := t.TempDir()
	r := &swRepo{t: t, dir: dir}
	r.git("init", "-q", "-b", "main")
	return r
}

func (r *swRepo) enter() {
	r.t.Helper()
	clearRepoEnv(r.t)
	r.t.Chdir(r.dir)
	git.Use(git.RealRunner{})
	r.t.Cleanup(func() { git.Use(git.RealRunner{}) })
}

func (r *swRepo) git(args ...string) string {
	r.t.Helper()
	out, err := r.gitRaw(args...)
	if err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(out)
}

func (r *swRepo) gitAllow(args ...string) error {
	r.t.Helper()
	_, err := r.gitRaw(args...)
	return err
}

func (r *swRepo) gitRaw(args ...string) (string, error) {
	cmd := exec.Command(git.Executable(), args...)
	cmd.Dir = r.dir
	withoutRepoEnv(cmd)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (r *swRepo) mustConflict(args ...string) {
	r.t.Helper()
	out, err := r.gitRaw(args...)
	if err == nil {
		r.t.Fatalf("git %s succeeded\n%s", strings.Join(args, " "), out)
	}
}

func (r *swRepo) write(name, body string) {
	r.t.Helper()
	if err := os.WriteFile(filepath.Join(r.dir, name), []byte(body), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *swRepo) read(name string) string {
	r.t.Helper()
	b, err := os.ReadFile(filepath.Join(r.dir, name))
	if err != nil {
		r.t.Fatal(err)
	}
	return string(b)
}

func (r *swRepo) commit(msg, name, body string) {
	r.t.Helper()
	r.write(name, body)
	r.git("add", name)
	r.git("commit", "-q", "-m", msg)
}

func (r *swRepo) addRemote(name string, branches ...string) {
	r.t.Helper()
	r.pushRemoteFlag(name, true, branches...)
}

func (r *swRepo) pushRemote(name string, branches ...string) {
	r.t.Helper()
	r.pushRemoteFlag(name, false, branches...)
}

func (r *swRepo) pushRemoteFlag(name string, upstream bool, branches ...string) {
	r.t.Helper()
	bare := r.t.TempDir()
	cmd := exec.Command(git.Executable(), "init", "--bare", "-q", "-b", "main", bare)
	withoutRepoEnv(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		r.t.Fatalf("bare: %v\n%s", err, out)
	}
	r.git("remote", "add", name, bare)
	args := []string{"push", "-q"}
	if upstream {
		args = append(args, "-u")
	}
	args = append(args, name)
	args = append(args, branches...)
	r.git(args...)
}

func (r *swRepo) remoteURL(name string) string {
	r.t.Helper()
	return r.git("remote", "get-url", name)
}

func (r *swRepo) headRef() string {
	r.t.Helper()
	return r.git("symbolic-ref", "-q", "HEAD")
}

func (r *swRepo) rev(ref string) string {
	r.t.Helper()
	return r.git("rev-parse", ref)
}

func (r *swRepo) upstream(branch string) string {
	r.t.Helper()
	return r.git("rev-parse", "--abbrev-ref", branch+"@{upstream}")
}

func (r *swRepo) stashList() string {
	r.t.Helper()
	return r.git("stash", "list")
}

func (r *swRepo) sourceOf(branch string) string {
	r.t.Helper()
	s, _, err := memrepo.LoadFromCWD()
	if err != nil {
		r.t.Fatal(err)
	}
	return memrepo.BranchSource(s, branch)
}

func gitDir(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(git.Executable(), args...)
	cmd.Dir = dir
	withoutRepoEnv(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func withoutRepoEnv(cmd *exec.Cmd) {
	env := make([]string, 0, len(os.Environ()))
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GIT_DIR=") || strings.HasPrefix(e, "GIT_WORK_TREE=") || strings.HasPrefix(e, "GIT_INDEX_FILE=") {
			continue
		}
		env = append(env, e)
	}
	cmd.Env = env
}

func clearRepoEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE"} {
		val, ok := os.LookupEnv(key)
		if !ok {
			continue
		}
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Setenv(key, val) })
	}
}

func runSwitch(p prompt.Prompter, args ...string) error {
	cmd := newSwitchCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), p))
	return switchRun(cmd, args)
}

func runSwitchCmd(p prompt.Prompter, args ...string) error {
	cmd := newSwitchCommand()
	cmd.SetContext(prompt.WithPrompter(context.Background(), p))
	return cmd.RunE(cmd, args)
}

func withStderr(t *testing.T, fn func() error) (error, string) {
	t.Helper()
	old := os.Stderr
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = pw
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, pr)
		close(done)
	}()
	callErr := fn()
	os.Stderr = old
	_ = pw.Close()
	<-done
	return callErr, buf.String()
}

func writeSwitchHook(t *testing.T, repo, kind, marker, word string) {
	t.Helper()
	dir := filepath.Join(repo, ".config", "elegant-git", "hooks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf("#!/bin/bash\necho %s >> %q\nexit 1\n", word, marker)
	if err := os.WriteFile(filepath.Join(dir, "work-switch-"+kind), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func readMarker(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

type swPrompter struct {
	picks      []string
	pickN      int
	pickDefs   []string
	choices    [][]prompt.Choice
	lines      []string
	lineN      int
	stringDefs []string
}

func (p *swPrompter) String(_, def string) (string, error) {
	p.stringDefs = append(p.stringDefs, def)
	if p.lineN < len(p.lines) {
		v := p.lines[p.lineN]
		p.lineN++
		return v, nil
	}
	if def != "" {
		return def, nil
	}
	return "", fmt.Errorf("unexpected string")
}
func (p *swPrompter) Confirm(string, bool) (bool, error) {
	return false, fmt.Errorf("unexpected confirm")
}
func (p *swPrompter) Choose(string, []string) (int, error) {
	return -1, fmt.Errorf("unexpected choose")
}
func (p *swPrompter) Required(string, string) error { return fmt.Errorf("unexpected required") }
func (p *swPrompter) EditOrAccept(string, string) (string, error) {
	return "", fmt.Errorf("unexpected edit")
}
func (p *swPrompter) Optional(string, string) (string, error) {
	return "", fmt.Errorf("unexpected optional")
}
func (p *swPrompter) Closed(string, []string, string, bool) (string, error) {
	return "", fmt.Errorf("unexpected closed")
}
func (p *swPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, fmt.Errorf("unexpected batch")
}
func (p *swPrompter) Pick(_ string, choices []prompt.Choice, def string) (string, error) {
	p.choices = append(p.choices, choices)
	p.pickDefs = append(p.pickDefs, def)
	if p.pickN >= len(p.picks) {
		return "", fmt.Errorf("unexpected pick")
	}
	v := p.picks[p.pickN]
	p.pickN++
	return v, nil
}
