package config

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/bees-hive/elegant-git/internal/git"
	memrepo "github.com/bees-hive/elegant-git/internal/memory/repo"
)

func setupBranchSourceTest(t *testing.T) (*git.MemoryRunner, string) {
	t.Helper()
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	t.Setenv("ELEGANT_GIT_REPO_STATE_FILE", filepath.Join(gitDir, "elegant-git", "state.json"))
	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	git.Use(m)
	return m, gitDir
}

func stubRefExists(m *git.MemoryRunner, refs ...string) {
	for _, ref := range refs {
		m.Outputs["rev-parse --verify --quiet "+ref] = "abc123"
	}
}

func stubLocalBranchExists(m *git.MemoryRunner, branches ...string) {
	for _, branch := range branches {
		m.Outputs["rev-parse --verify --quiet --abbrev-ref --branches=refs/heads "+branch] = branch
	}
}

func TestBranchSourceBranch(t *testing.T) {
	m, gitDir := setupBranchSourceTest(t)
	stubRefExists(m, "develop")
	stubLocalBranchExists(m, "feature")
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{"feature": "develop"},
	}); err != nil {
		t.Fatal(err)
	}

	if got := BranchSourceBranch("feature"); got != "develop" {
		t.Fatalf("BranchSourceBranch(feature) = %q, want develop", got)
	}
	if got := BranchSourceBranch("other"); got != DefaultBranchDefault {
		t.Fatalf("BranchSourceBranch(other) = %q, want %q", got, DefaultBranchDefault)
	}
}

func TestBranchSourceBranchMissingRecordedRef(t *testing.T) {
	m, gitDir := setupBranchSourceTest(t)
	stubLocalBranchExists(m, "feature")
	m.FailOn["rev-parse --verify --quiet origin/deleted"] = fmt.Errorf("unknown revision")
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{"feature": "origin/deleted"},
	}); err != nil {
		t.Fatal(err)
	}

	if got := BranchSourceBranch("feature"); got != DefaultBranchDefault {
		t.Fatalf("BranchSourceBranch(feature) = %q, want %q", got, DefaultBranchDefault)
	}
	loaded, err := memrepo.Load(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := memrepo.BranchSource(loaded, "feature"); got != DefaultBranchDefault {
		t.Fatalf("saved source = %q, want %q", got, DefaultBranchDefault)
	}
}

func TestBranchSourceBranchClearsWhenLocalBranchMissing(t *testing.T) {
	m, gitDir := setupBranchSourceTest(t)
	stubRefExists(m, "develop")
	m.FailOn["rev-parse --verify --quiet --abbrev-ref --branches=refs/heads feature"] = fmt.Errorf("unknown revision")
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{"feature": "develop"},
	}); err != nil {
		t.Fatal(err)
	}

	if got := BranchSourceBranch("feature"); got != DefaultBranchDefault {
		t.Fatalf("BranchSourceBranch(feature) = %q, want %q", got, DefaultBranchDefault)
	}
	loaded, err := memrepo.Load(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := memrepo.BranchSource(loaded, "feature"); got != "" {
		t.Fatalf("saved source = %q, want empty", got)
	}
}

func TestFreshestBranchSourceBranch(t *testing.T) {
	m, gitDir := setupBranchSourceTest(t)
	stubRefExists(m, "develop", "origin/develop", "origin/release")
	stubLocalBranchExists(m, "feature", "remote-feature")
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{
			"feature":        "develop",
			"remote-feature": "origin/release",
		},
	}); err != nil {
		t.Fatal(err)
	}
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["for-each-ref refs/remotes/origin/release"] = "ref"
	git.Use(m)

	if got := FreshestBranchSourceBranch("feature"); got != "origin/develop" {
		t.Fatalf("FreshestBranchSourceBranch(feature) = %q, want origin/develop", got)
	}
	if got := FreshestBranchSourceBranch("remote-feature"); got != "origin/release" {
		t.Fatalf("FreshestBranchSourceBranch(remote-feature) = %q, want origin/release", got)
	}

	m.Repo.Remotes = nil
	if got := FreshestBranchSourceBranch("feature"); got != "develop" {
		t.Fatalf("FreshestBranchSourceBranch(feature) without remotes = %q, want develop", got)
	}
}

func TestFreshestBranchSourceBranchPrefersConfiguredUpstream(t *testing.T) {
	m, gitDir := setupBranchSourceTest(t)
	stubRefExists(m, "develop", "origin/develop", "upstream/develop")
	stubLocalBranchExists(m, "feature")
	m.Outputs["rev-parse --abbrev-ref develop@{upstream}"] = "upstream/develop"
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{"feature": "develop"},
	}); err != nil {
		t.Fatal(err)
	}
	m.Repo.Remotes = []string{"origin", "upstream"}
	git.Use(m)

	if got := FreshestBranchSourceBranch("feature"); got != "upstream/develop" {
		t.Fatalf("FreshestBranchSourceBranch(feature) = %q, want upstream/develop", got)
	}
}

func TestFreshestBranchSourceBranchFallsBackToRemoteThenLocal(t *testing.T) {
	m, gitDir := setupBranchSourceTest(t)
	stubRefExists(m, "develop", "origin/develop")
	stubLocalBranchExists(m, "feature")
	m.FailOn["rev-parse --abbrev-ref develop@{upstream}"] = fmt.Errorf("no upstream")
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{"feature": "develop"},
	}); err != nil {
		t.Fatal(err)
	}
	m.Repo.Remotes = []string{"origin"}
	git.Use(m)

	if got := FreshestBranchSourceBranch("feature"); got != "origin/develop" {
		t.Fatalf("with remotes: got %q, want origin/develop", got)
	}

	m.Repo.Remotes = nil
	m.FailOn["rev-parse --verify --quiet origin/develop"] = fmt.Errorf("unknown revision")
	git.Use(m)

	if got := FreshestBranchSourceBranch("feature"); got != "develop" {
		t.Fatalf("without remote ref: got %q, want develop", got)
	}
}

func TestFreshestBranchSourceBranchMissingRemoteUsesDefault(t *testing.T) {
	m, gitDir := setupBranchSourceTest(t)
	stubRefExists(m, "main", "origin/main")
	stubLocalBranchExists(m, "feature")
	m.FailOn["rev-parse --verify --quiet origin/OPS-2162"] = fmt.Errorf("unknown revision")
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{"feature": "origin/OPS-2162"},
	}); err != nil {
		t.Fatal(err)
	}
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["for-each-ref refs/remotes/origin/OPS-2162"] = ""
	git.Use(m)

	if got := FreshestBranchSourceBranch("feature"); got != "origin/main" {
		t.Fatalf("FreshestBranchSourceBranch(feature) = %q, want origin/main", got)
	}
	loaded, err := memrepo.Load(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := memrepo.BranchSource(loaded, "feature"); got != DefaultBranchDefault {
		t.Fatalf("saved source = %q, want %q", got, DefaultBranchDefault)
	}
}

func TestSetBranchSourceBranch(t *testing.T) {
	_, gitDir := setupBranchSourceTest(t)
	if err := SetBranchSourceBranch("task-1", "origin/main"); err != nil {
		t.Fatal(err)
	}
	loaded, err := memrepo.Load(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := memrepo.BranchSource(loaded, "task-1"); got != "origin/main" {
		t.Fatalf("BranchSource = %q, want origin/main", got)
	}
}
