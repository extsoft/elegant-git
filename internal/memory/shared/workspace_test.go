package shared

import (
	"strings"
	"testing"
)

func TestCreateWorkspaceRejectsReservedNames(t *testing.T) {
	s := emptyState()
	for _, name := range []string{SelectorAll, SelectorCurrent, " all ", "\tcurrent"} {
		_, err := CreateWorkspace(s, CreateWorkspaceInput{Name: name, UserName: "U", UserEmail: "u@e.com"})
		if err == nil {
			t.Fatalf("name %q: expected error", name)
		}
		if !strings.Contains(err.Error(), "reserved") {
			t.Fatalf("name %q: got %v", name, err)
		}
	}
}

func TestDeleteWorkspaceUnlinksRepos(t *testing.T) {
	s := emptyState()
	id, err := CreateWorkspace(s, CreateWorkspaceInput{Name: "p", UserName: "U", UserEmail: "u@e.com"})
	if err != nil {
		t.Fatal(err)
	}
	s.Repositories["r1"] = &Repository{Name: "r1", WorkspaceID: id, CurrentPath: "/r1"}
	s.Workspaces[id].LinkedRepos = []string{"r1"}
	if err := DeleteWorkspace(s, id); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Workspaces[id]; ok {
		t.Fatal("workspace should be gone")
	}
	if s.Repositories["r1"].WorkspaceID != "" {
		t.Fatalf("workspace_id=%q want empty", s.Repositories["r1"].WorkspaceID)
	}
	if err := Validate(s); err != nil {
		t.Fatal(err)
	}
}

func TestAddWorkspaceNamespaceIdempotent(t *testing.T) {
	s := emptyState()
	id, err := CreateWorkspace(s, CreateWorkspaceInput{Name: "p", UserName: "U", UserEmail: "u@e.com"})
	if err != nil {
		t.Fatal(err)
	}
	changed, err := AddWorkspaceNamespace(s, id, "github.com/acme")
	if err != nil || !changed {
		t.Fatalf("first add: changed=%v err=%v", changed, err)
	}
	changed, err = AddWorkspaceNamespace(s, id, "github.com/acme")
	if err != nil || changed {
		t.Fatalf("second add: changed=%v err=%v", changed, err)
	}
	if got := s.Workspaces[id].Namespaces; len(got) != 1 || got[0] != "github.com/acme" {
		t.Fatalf("namespaces=%v", got)
	}
}

func TestFindWorkspacesByNamespace(t *testing.T) {
	s := emptyState()
	idB, _ := CreateWorkspace(s, CreateWorkspaceInput{
		Name: "beta", UserName: "U", UserEmail: "b@e.com", Namespaces: []string{"github.com/acme"},
	})
	idA, _ := CreateWorkspace(s, CreateWorkspaceInput{
		Name: "alpha", UserName: "U", UserEmail: "a@e.com", Namespaces: []string{"github.com/acme", "gitlab.com/x"},
	})
	_, _ = CreateWorkspace(s, CreateWorkspaceInput{Name: "gamma", UserName: "U", UserEmail: "g@e.com"})

	got := FindWorkspacesByNamespace(s, "github.com/acme")
	if len(got) != 2 || got[0] != idA || got[1] != idB {
		t.Fatalf("got %v want [%s %s]", got, idA, idB)
	}
	if got := FindWorkspacesByNamespace(s, "gitlab.com/x"); len(got) != 1 || got[0] != idA {
		t.Fatalf("gitlab hit: %v", got)
	}
	if got := FindWorkspacesByNamespace(s, "missing"); len(got) != 0 {
		t.Fatalf("missing: %v", got)
	}
}
