package shared

import "testing"

func TestDeleteProfileRefusesWhenLinked(t *testing.T) {
	s := emptyState()
	id, err := CreateProfile(s, CreateProfileInput{Name: "work", UserName: "A", UserEmail: "a@b.c"})
	if err != nil {
		t.Fatal(err)
	}
	s.Profiles[id].LinkedRepos = []string{"repo-1"}
	if err := DeleteProfile(s, id); err == nil {
		t.Fatal("expected error")
	}
}
