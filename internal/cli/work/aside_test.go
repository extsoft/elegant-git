package work

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
)

func TestCollidingPathsFiltersTrackedAndMissing(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "third_party", "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "third_party", "p", "CHANGELOG.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "from-target.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --show-toplevel"] = root
	if err := os.WriteFile(filepath.Join(root, "café.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Outputs["log -z --format= --name-only --diff-filter=A --no-renames --no-merges base..HEAD"] = "third_party/p/CHANGELOG.md\x00tracked.txt\x00missing.txt\x00café.txt\x00"
	m.Outputs["diff -z --name-only --diff-filter=A --no-renames HEAD main"] = "from-target.txt\x00"
	m.Outputs["ls-files -z"] = "tracked.txt\x00"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	got, err := collidingPaths("base", "main")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"café.txt", "from-target.txt", "third_party/p/CHANGELOG.md"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMoveAsideRollsBackWhenALaterRenameFails(t *testing.T) {
	root := t.TempDir()
	gitDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	aside := filepath.Join(gitDir, "elegant-git", "aside")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(aside, "sub")), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aside, "sub"), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --show-toplevel"] = root
	m.Outputs["rev-parse --git-dir"] = gitDir
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	err := moveAside([]string{"a.txt", "sub/b.txt"})
	if err == nil {
		t.Fatal("expected rename to fail")
	}
	if _, statErr := os.Stat(filepath.Join(root, "a.txt")); statErr != nil {
		t.Fatalf("a.txt was not restored: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(aside, "a.txt")); !os.IsNotExist(statErr) {
		t.Fatal("a.txt stayed aside")
	}
}

func TestRestoreAsideKeepsOccupiedDestination(t *testing.T) {
	root := t.TempDir()
	gitDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "b.txt"), []byte("live\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	aside := filepath.Join(gitDir, "elegant-git", "aside", "sub")
	if err := os.MkdirAll(aside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aside, "b.txt"), []byte("backup\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --show-toplevel"] = root
	m.Outputs["rev-parse --git-dir"] = gitDir
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	err := restoreAside()
	if err == nil || !strings.Contains(err.Error(), "aside copy was kept at") {
		t.Fatalf("err=%v", err)
	}
	live, readErr := os.ReadFile(filepath.Join(root, "sub", "b.txt"))
	if readErr != nil || string(live) != "live\n" {
		t.Fatalf("destination %q err=%v", live, readErr)
	}
	kept, readErr := os.ReadFile(filepath.Join(aside, "b.txt"))
	if readErr != nil || string(kept) != "backup\n" {
		t.Fatalf("aside %q err=%v", kept, readErr)
	}
}

func TestRestoreAsideLeavesEarlierFilesWhenALaterDestinationExists(t *testing.T) {
	root := t.TempDir()
	gitDir := t.TempDir()
	aside := filepath.Join(gitDir, "elegant-git", "aside")
	if err := os.MkdirAll(aside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aside, "a.txt"), []byte("aside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(aside, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aside, "sub", "b.txt"), []byte("aside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "b.txt"), []byte("live\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --show-toplevel"] = root
	m.Outputs["rev-parse --git-dir"] = gitDir
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	if err := restoreAside(); err == nil {
		t.Fatal("expected occupied destination to be refused")
	}
	if _, err := os.Stat(filepath.Join(root, "a.txt")); !os.IsNotExist(err) {
		t.Fatalf("a.txt was restored despite a later collision: %v", err)
	}
	if _, err := os.Stat(filepath.Join(aside, "a.txt")); err != nil {
		t.Fatalf("a.txt left the aside: %v", err)
	}
}
