package workspace

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/shared"
)

func TestFetchRunUsesLinkedRepos(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	repoA := filepath.Join(dir, "a")
	repoB := filepath.Join(dir, "b")
	for _, p := range []string{repoA, repoB} {
		if err := os.MkdirAll(filepath.Join(p, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	m := git.NewMemoryRunner()
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["fetch --all --tags --prune"] = "From github.com:acme/a"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{ID: "ra", Name: "a", WorkspaceID: id, CurrentPath: repoA})
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{ID: "rb", Name: "b", WorkspaceID: id, CurrentPath: repoB})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := fetchRun(&buf, "acme"); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, c := range m.Calls {
		if strings.Join(c.Args, " ") == "fetch --all --tags --prune" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("fetch calls=%d calls=%v", count, m.Calls)
	}
	out := buf.String()
	for _, want := range []string{
		"[1/2] fetching a",
		"  From github.com:acme/a",
		"ok   a",
		"[2/2] fetching b",
		"ok   b",
		"Fetched: 2 | Skipped: 0 | Failed: 0",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestFetchRunResolvesFromCurrentRepo(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	repoA := filepath.Join(dir, "a")
	if err := os.MkdirAll(filepath.Join(repoA, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["rev-parse --git-dir"] = filepath.Join(repoA, ".git")
	m.Repo.LocalConfig["elegant-git.repo-id"] = "ra"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{ID: "ra", Name: "a", WorkspaceID: id, CurrentPath: repoA})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := fetchRun(&buf, ""); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range m.Calls {
		if strings.Join(c.Args, " ") == "fetch --all --tags --prune" {
			found = true
		}
	}
	if !found {
		t.Fatalf("calls=%v", m.Calls)
	}
}

func TestFetchRunKeepsGitError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	repoA := filepath.Join(dir, "a")
	if err := os.MkdirAll(filepath.Join(repoA, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Repo.Remotes = []string{"origin"}
	m.FailOn["fetch --all --tags --prune"] = fmt.Errorf("exit status 1")
	m.Outputs["fetch --all --tags --prune"] = "fatal: unable to access 'https://example.com/a.git/': Could not resolve host"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{ID: "ra", Name: "a", WorkspaceID: id, CurrentPath: repoA})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err = fetchRun(&buf, "acme")
	if err == nil {
		t.Fatal("expected error when fetch fails")
	}
	if !strings.Contains(err.Error(), "fetch failed for 1 of 1") {
		t.Fatalf("err=%v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"fatal: unable to access 'https://example.com/a.git/': Could not resolve host",
		"fail a",
		"Fetched: 0 | Skipped: 0 | Failed: 1",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "exit status 1") {
		t.Errorf("generic exit status leaked into:\n%s", out)
	}
}

func TestFetchRunSuggestsDoctorOnPathMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	missing := filepath.Join(dir, "active-sales")

	m := git.NewMemoryRunner()
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{
		ID: "ra", Name: "active-sales", WorkspaceID: id, CurrentPath: missing,
	})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err = fetchRun(&buf, "acme")
	if err == nil {
		t.Fatal("expected error")
	}
	out := buf.String()
	for _, want := range []string{
		"path missing: " + missing,
		"fail active-sales",
		"Run `git elegant workspace doctor acme` to repair missing paths from the workspace side,",
		"or `cd` into a relocated clone and run `git elegant repo doctor`.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestFetchRunSkipNoRemotes(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	repoA := filepath.Join(dir, "a")
	if err := os.MkdirAll(filepath.Join(repoA, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: "acme", UserName: "U", UserEmail: "u@e.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = shared.UpsertRepo(s, shared.UpsertRepoInput{ID: "ra", Name: "a", WorkspaceID: id, CurrentPath: repoA})
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := fetchRun(&buf, "acme"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"skip a", "Fetched: 0 | Skipped: 1 | Failed: 0"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	for _, c := range m.Calls {
		if len(c.Args) > 0 && c.Args[0] == "fetch" {
			t.Fatalf("unexpected fetch: %v", c.Args)
		}
	}
}

func TestFetchDisplayKeepsFailLogs(t *testing.T) {
	var buf bytes.Buffer
	d := newFetchDisplay(&buf, 1)
	d.start("lib")
	d.log("fatal: unable to access remote")
	d.finish("lib", "fail", "exit status 1")
	got := strings.Join(d.lines(), "\n")
	for _, want := range []string{"fail lib", "fatal: unable to access remote"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "exit status 1") {
		t.Errorf("generic exit status leaked into:\n%s", got)
	}
}

func TestFetchDisplayLines(t *testing.T) {
	d := &fetchDisplay{total: 3, done: 1, current: "lib", items: []fetchItem{
		{name: "app", status: "ok"},
	}, logs: []string{"From github.com:acme/lib"}}
	got := strings.Join(d.lines(), "\n")
	for _, want := range []string{
		"1/3  fetching lib",
		"  ok   app",
		"  .... lib",
		"       From github.com:acme/lib",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "[") || !strings.Contains(got, "]") {
		t.Fatalf("missing progress bar: %s", got)
	}
}

func TestProgressBar(t *testing.T) {
	if got := progressBar(0, 4, 4); got != "[>   ]" {
		t.Fatalf("empty: %q", got)
	}
	if got := progressBar(2, 4, 4); got != "[==> ]" {
		t.Fatalf("half: %q", got)
	}
	if got := progressBar(4, 4, 4); got != "[====]" {
		t.Fatalf("full: %q", got)
	}
}

func TestClipLine(t *testing.T) {
	if got := clipLine("hello", 0); got != "hello" {
		t.Fatalf("width 0: %q", got)
	}
	if got := clipLine("hello", 10); got != "hello" {
		t.Fatalf("short: %q", got)
	}
	if got := clipLine("hello world", 8); got != "hello w…" {
		t.Fatalf("clip: %q", got)
	}
}
