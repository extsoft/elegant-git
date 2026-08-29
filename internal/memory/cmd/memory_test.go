package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/git"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	s := State{
		"work.accept": {Stash: "msg", Branch: "feat"},
	}
	if err := Save(gitDir, s); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded["work.accept"].Stash != "msg" || loaded["work.accept"].Branch != "feat" {
		t.Fatalf("got %+v", loaded["work.accept"])
	}
}

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	s, err := Load(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 0 {
		t.Fatalf("expected empty, got %v", s)
	}
}

func TestMultipleCommandsIsolated(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	s := State{
		"work.accept": {Branch: "a"},
		"release.new": {Branch: "b"},
	}
	if err := Save(gitDir, s); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded["work.accept"].Branch != "a" || loaded["release.new"].Branch != "b" {
		t.Fatal(loaded)
	}
}

func TestUnsetRemovesEntryKeepsFile(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	git.Use(m)

	s := State{"work.start": {Stash: "only"}}
	if err := Save(gitDir, s); err != nil {
		t.Fatal(err)
	}

	id := cmdid.ID{Command: "work", Action: "start"}
	if err := Unset(id, FieldStash); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected empty map, got %v", loaded)
	}
	if _, err := os.Stat(Path(gitDir)); err != nil {
		t.Fatalf("commands file should exist: %v", err)
	}
}

func TestGetSetRoundTrip(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = gitDir
	git.Use(m)

	id := cmdid.ID{Command: "work", Action: "push"}
	if err := Set(id, FieldBranch, "feat"); err != nil {
		t.Fatal(err)
	}
	got, err := Get(id, FieldBranch)
	if err != nil || got != "feat" {
		t.Fatalf("Get=%q err=%v", got, err)
	}
}
