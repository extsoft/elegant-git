package shared

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateProfileAndRepoLink(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))

	s := emptyState()
	id, err := CreateProfile(s, CreateProfileInput{
		Name: "work", UserName: "Jane", UserEmail: "jane@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := UpsertRepo(s, UpsertRepoInput{
		ID: "repo-1", Name: "proj", ProfileID: id, CurrentPath: "/tmp/proj",
	}); err != nil {
		t.Fatal(err)
	}
	if err := Validate(s); err != nil {
		t.Fatal(err)
	}
	if len(s.Profiles[id].LinkedRepos) != 1 {
		t.Fatalf("linked_repos = %v", s.Profiles[id].LinkedRepos)
	}
	if err := Save(s); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Profiles[id].Name != "work" {
		t.Fatalf("got name %q", loaded.Profiles[id].Name)
	}
}

func TestRelinkUpdatesIndex(t *testing.T) {
	s := emptyState()
	p1, _ := CreateProfile(s, CreateProfileInput{Name: "a", UserName: "A", UserEmail: "a@test"})
	p2, _ := CreateProfile(s, CreateProfileInput{Name: "b", UserName: "B", UserEmail: "b@test"})
	_ = UpsertRepo(s, UpsertRepoInput{ID: "r1", Name: "r", ProfileID: p1, CurrentPath: "/a"})
	if err := Relink(s, "r1", p2); err != nil {
		t.Fatal(err)
	}
	if containsString(s.Profiles[p1].LinkedRepos, "r1") {
		t.Fatal("old profile still linked")
	}
	if !containsString(s.Profiles[p2].LinkedRepos, "r1") {
		t.Fatal("new profile not linked")
	}
}

func TestDeleteProfileRequiresUnlink(t *testing.T) {
	s := emptyState()
	id, _ := CreateProfile(s, CreateProfileInput{Name: "x", UserName: "X", UserEmail: "x@test"})
	_ = UpsertRepo(s, UpsertRepoInput{ID: "r1", Name: "r", ProfileID: id, CurrentPath: "/a"})
	if err := DeleteProfile(s, id); err == nil {
		t.Fatal("expected error")
	}
	if err := DeleteRepo(s, "r1"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteProfile(s, id); err != nil {
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
