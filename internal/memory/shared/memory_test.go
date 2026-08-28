package shared

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateWorkspaceAndRepoLink(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))

	s := emptyState()
	id, err := CreateWorkspace(s, CreateWorkspaceInput{
		Name: "work", UserName: "Jane", UserEmail: "jane@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := UpsertRepo(s, UpsertRepoInput{
		ID: "repo-1", Name: "proj", WorkspaceID: id, CurrentPath: "/tmp/proj",
	}); err != nil {
		t.Fatal(err)
	}
	if err := Validate(s); err != nil {
		t.Fatal(err)
	}
	if len(s.Workspaces[id].LinkedRepos) != 1 {
		t.Fatalf("linked_repos = %v", s.Workspaces[id].LinkedRepos)
	}
	if err := Save(s); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Workspaces[id].Name != "work" {
		t.Fatalf("got name %q", loaded.Workspaces[id].Name)
	}
}

func TestRelinkUpdatesIndex(t *testing.T) {
	s := emptyState()
	p1, _ := CreateWorkspace(s, CreateWorkspaceInput{Name: "a", UserName: "A", UserEmail: "a@test"})
	p2, _ := CreateWorkspace(s, CreateWorkspaceInput{Name: "b", UserName: "B", UserEmail: "b@test"})
	_ = UpsertRepo(s, UpsertRepoInput{ID: "r1", Name: "r", WorkspaceID: p1, CurrentPath: "/a"})
	if err := Relink(s, "r1", p2); err != nil {
		t.Fatal(err)
	}
	if containsString(s.Workspaces[p1].LinkedRepos, "r1") {
		t.Fatal("old workspace still linked")
	}
	if !containsString(s.Workspaces[p2].LinkedRepos, "r1") {
		t.Fatal("new workspace not linked")
	}
}

func TestDeleteWorkspaceRequiresUnlink(t *testing.T) {
	s := emptyState()
	id, _ := CreateWorkspace(s, CreateWorkspaceInput{Name: "x", UserName: "X", UserEmail: "x@test"})
	_ = UpsertRepo(s, UpsertRepoInput{ID: "r1", Name: "r", WorkspaceID: id, CurrentPath: "/a"})
	if err := DeleteWorkspace(s, id); err == nil {
		t.Fatal("expected error")
	}
	if err := DeleteRepo(s, "r1"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteWorkspace(s, id); err != nil {
		t.Fatal(err)
	}
}

func TestPath(t *testing.T) {
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(t.TempDir(), "s.json"))
	p, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(p)); os.IsNotExist(err) {
		// dir may not exist until save
	}
}
