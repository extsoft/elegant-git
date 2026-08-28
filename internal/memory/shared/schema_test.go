package shared_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bees-hive/elegant-git/internal/deprecation"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
)

func TestLoadAutoMigratesV1AndBacksUp(t *testing.T) {
	deprecation.Reset()
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	t.Setenv("ELEGANT_GIT_STATE_FILE", path)

	v1 := `{
  "schema_version": 1,
  "profiles": {
    "ws-1": {
      "name": "work",
      "user_name": "Jane",
      "user_email": "jane@example.com",
      "linked_repos": ["repo-1"]
    }
  },
  "repositories": {
    "repo-1": {
      "name": "proj",
      "profile_id": "ws-1",
      "current_path": "/tmp/proj"
    }
  }
}`
	if err := os.WriteFile(path, []byte(v1), 0o600); err != nil {
		t.Fatal(err)
	}

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	if s.SchemaVersion != shared.SchemaVersion {
		t.Fatalf("schema_version = %d", s.SchemaVersion)
	}
	ws, ok := s.Workspaces["ws-1"]
	if !ok || ws == nil || ws.Name != "work" {
		t.Fatalf("workspaces = %#v", s.Workspaces)
	}
	repo := s.Repositories["repo-1"]
	if repo == nil || repo.WorkspaceID != "ws-1" {
		t.Fatalf("repositories = %#v", s.Repositories)
	}

	bak := path + ".bak"
	bakData, err := os.ReadFile(bak)
	if err != nil {
		t.Fatalf("expected backup at %s: %v", bak, err)
	}
	if !strings.Contains(string(bakData), `"profiles"`) {
		t.Fatalf("backup should retain v1 content: %s", bakData)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(data)
	if strings.Contains(raw, `"profiles"`) || strings.Contains(raw, `"profile_id"`) {
		t.Fatalf("migrated file still has legacy keys: %s", raw)
	}
	if !strings.Contains(raw, `"workspaces"`) || !strings.Contains(raw, `"workspace_id"`) {
		t.Fatalf("migrated file missing v2 keys: %s", raw)
	}

	events := deprecation.Events()
	found := false
	for _, e := range events {
		if e.ID == deprecation.DEP012 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected DEP-012, got %#v", events)
	}
}

func TestSaveWritesOnlyV2Keys(t *testing.T) {
	deprecation.Reset()
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	t.Setenv("ELEGANT_GIT_STATE_FILE", path)

	s := &shared.State{
		SchemaVersion: shared.SchemaVersion,
		Workspaces: map[string]*shared.Workspace{
			"ws-1": {Name: "work", UserName: "Jane", UserEmail: "jane@example.com", LinkedRepos: []string{}},
		},
		Repositories: map[string]*shared.Repository{},
	}
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(data)
	if strings.Contains(raw, `"profiles"`) || strings.Contains(raw, `"profile_id"`) {
		t.Fatalf("legacy keys present: %s", raw)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded["workspaces"]; !ok {
		t.Fatal("missing workspaces key")
	}
	var ver int
	if err := json.Unmarshal(decoded["schema_version"], &ver); err != nil {
		t.Fatal(err)
	}
	if ver != 2 {
		t.Fatalf("schema_version = %d", ver)
	}
}

func TestLoadPrefersNewKeysOverLegacy(t *testing.T) {
	deprecation.Reset()
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	t.Setenv("ELEGANT_GIT_STATE_FILE", path)

	mixed := `{
  "schema_version": 1,
  "workspaces": {
    "new-1": {"name": "new", "user_name": "N", "user_email": "n@e", "linked_repos": []}
  },
  "profiles": {
    "old-1": {"name": "old", "user_name": "O", "user_email": "o@e", "linked_repos": []}
  },
  "repositories": {
    "r1": {"name": "r", "workspace_id": "new-1", "profile_id": "old-1", "current_path": "/r"}
  }
}`
	if err := os.WriteFile(path, []byte(mixed), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Workspaces["new-1"]; !ok {
		t.Fatal("expected new workspace")
	}
	if _, ok := s.Workspaces["old-1"]; ok {
		t.Fatal("should not fall back to profiles when workspaces present")
	}
	if s.Repositories["r1"].WorkspaceID != "new-1" {
		t.Fatalf("workspace_id = %q", s.Repositories["r1"].WorkspaceID)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("expected backup: %v", err)
	}
}

func TestLoadRejectsNewerSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	t.Setenv("ELEGANT_GIT_STATE_FILE", path)
	if err := os.WriteFile(path, []byte(`{"schema_version":99,"workspaces":{},"repositories":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := shared.Load()
	if err == nil || !strings.Contains(err.Error(), "newer than supported") {
		t.Fatalf("got %v", err)
	}
}
