package work

import (
	"bytes"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
)

func TestListRunBranchOnly(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Repo.CurrentBranch = "feature"
	git.Use(m)

	var buf bytes.Buffer
	if err := listRun(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Branch\nlocal:  feature") {
		t.Fatalf("got:\n%s", out)
	}
	if !strings.Contains(out, "remote: none") {
		t.Fatalf("got:\n%s", out)
	}
	if strings.Contains(out, "Further steps:") {
		t.Fatalf("no extra sections, got:\n%s", out)
	}
}

func TestListRunModificationsFurtherSteps(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Repo.CurrentBranch = "feature"
	m.Outputs["status --short"] = "M  staged.go\n M unstaged.go"
	git.Use(m)

	var buf bytes.Buffer
	if err := listRun(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Modifications\nM  staged.go\n M unstaged.go\n") {
		t.Fatalf("got:\n%s", out)
	}
	if !strings.Contains(out, "Further steps:") || !strings.Contains(out, "eg work save") {
		t.Fatalf("got:\n%s", out)
	}
	if strings.Contains(out, "eg work polish") {
		t.Fatalf("no unique commits, got:\n%s", out)
	}
}

func TestGitColorArgs(t *testing.T) {
	got := gitColorArgs(false, "status", "--short")
	if strings.Join(got, " ") != "status --short" {
		t.Fatalf("plain: %q", got)
	}
	got = gitColorArgs(true, "status", "--short")
	if strings.Join(got, " ") != "-c color.ui=always status --short" {
		t.Fatalf("color: %q", got)
	}
}

func TestListRunModificationsBeforeCommits(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Repo.CurrentBranch = "feature"
	m.Outputs["status --short"] = " M file.go"
	m.Outputs["rev-list main..feature"] = "abc"
	m.Outputs["log --oneline main..feature"] = "abc commit"
	git.Use(m)

	var buf bytes.Buffer
	if err := listRun(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	mod := strings.Index(out, "\nModifications\n")
	commits := strings.Index(out, "\nCommits")
	if mod < 0 || commits < 0 || mod > commits {
		t.Fatalf("modifications should precede commits, got:\n%s", out)
	}
	save := strings.Index(out, "eg work save")
	polish := strings.Index(out, "eg work polish")
	if save < 0 || polish < 0 || save > polish {
		t.Fatalf("save should precede polish, got:\n%s", out)
	}
}
