package shared

import "testing"

func TestDeleteWorkspaceRefusesWhenLinked(t *testing.T) {
	s := emptyState()
	id, err := CreateWorkspace(s, CreateWorkspaceInput{Name: "p", UserName: "U", UserEmail: "u@e.com"})
	if err != nil {
		t.Fatal(err)
	}
	s.Workspaces[id].LinkedRepos = []string{"r1"}
	if err := DeleteWorkspace(s, id); err == nil {
		t.Fatal("expected error when linked")
	}
}
