package shared

import "testing"

func TestResolveRepositoryByName(t *testing.T) {
	s := emptyState()
	pID, _ := CreateProfile(s, CreateProfileInput{Name: "p", UserName: "U", UserEmail: "u@e.com"})
	s.Repositories["r1"] = &Repository{Name: "myapp", ProfileID: pID, CurrentPath: "/tmp/myapp"}
	id, r, err := ResolveRepository(s, "myapp")
	if err != nil || id != "r1" || r.Name != "myapp" {
		t.Fatalf("got id=%q repo=%v err=%v", id, r, err)
	}
}

func TestResolveRepositoryByPath(t *testing.T) {
	s := emptyState()
	pID, _ := CreateProfile(s, CreateProfileInput{Name: "p", UserName: "U", UserEmail: "u@e.com"})
	s.Repositories["r1"] = &Repository{Name: "myapp", ProfileID: pID, CurrentPath: "/tmp/myapp"}
	_, r, err := ResolveRepository(s, "/tmp/myapp")
	if err != nil || r.CurrentPath != "/tmp/myapp" {
		t.Fatalf("got %v err=%v", r, err)
	}
}
