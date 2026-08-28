package repo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	s := &State{
		RepoID: "id-1", WorkspaceID: "ws-1",
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
	if loaded.WorkspaceID != "ws-1" {
		t.Fatalf("workspace %q", loaded.WorkspaceID)
	}
}

func TestLoadMigratesV1ProfileID(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(filepath.Join(gitDir, dirName), 0o755); err != nil {
		t.Fatal(err)
	}
	path := Path(gitDir)
	v1 := `{"schema_version":1,"repo_id":"r1","profile_id":"old-ws","default_branch":"main","protected_branches":["main"]}`
	if err := os.WriteFile(path, []byte(v1), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.WorkspaceID != "old-ws" {
		t.Fatalf("workspace_id = %q", loaded.WorkspaceID)
	}
	if loaded.SchemaVersion != SchemaVersion {
		t.Fatalf("schema = %d", loaded.SchemaVersion)
	}
	bak, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("expected backup: %v", err)
	}
	if !strings.Contains(string(bak), `"profile_id"`) {
		t.Fatalf("backup should retain v1: %s", bak)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"profile_id"`) {
		t.Fatalf("legacy key still present: %s", data)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded["workspace_id"]; !ok {
		t.Fatal("missing workspace_id")
	}
}

func TestBranchSourcesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	s := &State{BranchSources: map[string]string{"feature": "origin/main"}}
	SetBranchSource(s, "task", "develop")
	if err := Save(gitDir, s); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := BranchSource(loaded, "feature"); got != "origin/main" {
		t.Fatalf("feature source %q", got)
	}
	if got := BranchSource(loaded, "task"); got != "develop" {
		t.Fatalf("task source %q", got)
	}
	ClearBranchSource(loaded, "task")
	if got := BranchSource(loaded, "task"); got != "" {
		t.Fatalf("cleared source %q", got)
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
