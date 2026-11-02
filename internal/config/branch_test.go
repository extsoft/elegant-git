package config

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
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

func TestFreshestBranchSourceBranchGoneUpstreamUsesDefault(t *testing.T) {
	m, gitDir := setupBranchSourceTest(t)
	stubRefExists(m, "main", "origin/main", "A")
	stubLocalBranchExists(m, "B", "A")
	m.Repo.Remotes = []string{"origin"}
	m.Repo.LocalConfig["branch.A.merge"] = "refs/heads/A"
	m.FailOn["rev-parse --abbrev-ref A@{upstream}"] = fmt.Errorf("gone")
	m.FailOn["rev-parse --verify --quiet origin/A"] = fmt.Errorf("unknown")
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{"B": "A"},
	}); err != nil {
		t.Fatal(err)
	}

	if got := FreshestBranchSourceBranch("B"); got != "origin/main" {
		t.Fatalf("got %q, want origin/main", got)
	}
}

func TestFreshestBranchSourceBranchContainedUsesDefault(t *testing.T) {
	m, gitDir := setupBranchSourceTest(t)
	stubRefExists(m, "main", "origin/main", "A", "origin/A")
	stubLocalBranchExists(m, "B", "A")
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["rev-parse --abbrev-ref A@{upstream}"] = "origin/A"
	m.Outputs["rev-parse A"] = "aaa"
	m.Outputs["rev-parse origin/A"] = "aaa"
	m.Outputs["merge-base origin/main origin/A"] = "o"
	m.Outputs["rev-parse origin/main^{tree}"] = "S"
	m.Outputs["merge-tree --write-tree --merge-base=o origin/main origin/A"] = "S"
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{"B": "A"},
	}); err != nil {
		t.Fatal(err)
	}

	if got := FreshestBranchSourceBranch("B"); got != "origin/main" {
		t.Fatalf("got %q, want origin/main", got)
	}
}

func TestFreshestBranchSourceBranchPrefersLocalAheadOfRemote(t *testing.T) {
	m, gitDir := setupBranchSourceTest(t)
	stubRefExists(m, "main", "origin/main", "A", "origin/A")
	stubLocalBranchExists(m, "B", "A")
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["rev-parse --abbrev-ref A@{upstream}"] = "origin/A"
	m.Outputs["rev-parse A"] = "localsha"
	m.Outputs["rev-parse origin/A"] = "remotesha"
	m.Outputs["merge-base A origin/A"] = "o"
	m.Outputs["rev-parse A^{tree}"] = "localtree"
	m.Outputs["merge-tree --write-tree --merge-base=o A origin/A"] = "localtree"
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{"B": "A"},
	}); err != nil {
		t.Fatal(err)
	}

	if got := FreshestBranchSourceBranch("B"); got != "A" {
		t.Fatalf("got %q, want A", got)
	}
}

func TestFreshestBranchSourceBranchKeepsSlashedLocalName(t *testing.T) {
	m, gitDir := setupBranchSourceTest(t)
	stubRefExists(m, "main", "origin/main", "feature/login", "origin/login", "login")
	stubLocalBranchExists(m, "B", "feature/login", "login")
	m.Repo.Remotes = []string{"origin"}
	m.FailOn["rev-parse --verify --quiet origin/feature/login"] = fmt.Errorf("unknown")
	m.FailOn["rev-parse --abbrev-ref feature/login@{upstream}"] = fmt.Errorf("no upstream")
	m.FailOn["rev-parse --abbrev-ref login@{upstream}"] = fmt.Errorf("no upstream")
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{"B": "feature/login"},
	}); err != nil {
		t.Fatal(err)
	}

	if got := FreshestBranchSourceBranch("B"); got != "feature/login" {
		t.Fatalf("got %q, want feature/login", got)
	}
}

func TestFreshestBranchSourceBranchDoesNotPreferWrongSlashSegment(t *testing.T) {
	m, gitDir := setupBranchSourceTest(t)
	stubRefExists(m, "main", "origin/main", "feature/login", "login")
	stubLocalBranchExists(m, "B", "feature/login", "login")
	m.Repo.Remotes = []string{"origin"}
	m.FailOn["rev-parse --verify --quiet origin/feature/login"] = fmt.Errorf("unknown")
	m.FailOn["rev-parse --verify --quiet origin/login"] = fmt.Errorf("unknown")
	m.FailOn["rev-parse --abbrev-ref feature/login@{upstream}"] = fmt.Errorf("no upstream")
	m.FailOn["rev-parse --abbrev-ref login@{upstream}"] = fmt.Errorf("no upstream")
	m.Outputs["rev-parse login"] = "loginsha"
	m.Outputs["rev-parse feature/login"] = "featuresha"
	m.Outputs["merge-base login feature/login"] = "o"
	m.Outputs["rev-parse login^{tree}"] = "ltree"
	m.Outputs["merge-tree --write-tree --merge-base=o login feature/login"] = "ltree"
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{"B": "feature/login"},
	}); err != nil {
		t.Fatal(err)
	}

	if got := FreshestBranchSourceBranch("B"); got != "feature/login" {
		t.Fatalf("got %q, want feature/login", got)
	}
}

func TestFreshestBranchSourceBranchGoneUpstreamUsesSlashedName(t *testing.T) {
	m, gitDir := setupBranchSourceTest(t)
	stubRefExists(m, "main", "origin/main", "feature/login", "origin/login", "login")
	stubLocalBranchExists(m, "B", "feature/login", "login")
	m.Repo.Remotes = []string{"origin"}
	m.Repo.LocalConfig["branch.feature/login.merge"] = "refs/heads/feature/login"
	m.FailOn["rev-parse --abbrev-ref feature/login@{upstream}"] = fmt.Errorf("gone")
	m.FailOn["rev-parse --verify --quiet origin/feature/login"] = fmt.Errorf("unknown")
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{"B": "feature/login"},
	}); err != nil {
		t.Fatal(err)
	}

	if got := FreshestBranchSourceBranch("B"); got != "origin/main" {
		t.Fatalf("got %q, want origin/main", got)
	}
}

func TestFreshestBranchSourceBranchKeepsRemoteWhenLocalMatches(t *testing.T) {
	m, gitDir := setupBranchSourceTest(t)
	stubRefExists(m, "main", "origin/main", "A", "origin/A")
	stubLocalBranchExists(m, "B", "A")
	m.Repo.Remotes = []string{"origin"}
	m.Outputs["rev-parse --abbrev-ref A@{upstream}"] = "origin/A"
	m.Outputs["rev-parse A"] = "aaa"
	m.Outputs["rev-parse origin/A"] = "aaa"
	if err := memrepo.Save(gitDir, &memrepo.State{
		BranchSources: map[string]string{"B": "A"},
	}); err != nil {
		t.Fatal(err)
	}

	if got := FreshestBranchSourceBranch("B"); got != "origin/A" {
		t.Fatalf("got %q, want origin/A", got)
	}
}
