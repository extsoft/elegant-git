package repo

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	s := &State{
		RepoID: "id-1", ProfileID: "prof-1",
		DefaultBranch: "main", ProtectedBranches: []string{"main"},
	}
	if err := Save(gitDir, s); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DefaultBranch != "main" {
		t.Fatalf("branch %q", loaded.DefaultBranch)
	}
	if loaded.ProfileID != "prof-1" {
		t.Fatalf("profile %q", loaded.ProfileID)
	}
}

func TestDefaultBranchName(t *testing.T) {
	if DefaultBranchName(nil) != defaultBranchDefault {
		t.Fatal()
	}
	if DefaultBranchName(&State{DefaultBranch: "dev"}) != "dev" {
		t.Fatal()
	}
}
