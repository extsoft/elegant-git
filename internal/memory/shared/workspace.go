package shared

import (
	"fmt"
	"strings"

	"github.com/bees-hive/elegant-git/internal/uuidv7"
)

// ListWorkspaces returns all workspaces keyed by id.
func ListWorkspaces(s *State) map[string]*Workspace {
	if s == nil || s.Workspaces == nil {
		return map[string]*Workspace{}
	}
	return s.Workspaces
}

// GetWorkspace returns a workspace by id.
func GetWorkspace(s *State, id string) (*Workspace, error) {
	ws, ok := s.Workspaces[id]
	if !ok || ws == nil {
		return nil, ErrNotFound
	}
	return ws, nil
}

// GetWorkspaceByName returns workspace id and workspace for a display name.
func GetWorkspaceByName(s *State, name string) (string, *Workspace, error) {
	for id, ws := range s.Workspaces {
		if ws != nil && ws.Name == name {
			return id, ws, nil
		}
	}
	return "", nil, ErrNotFound
}

// GetWorkspaceByUserEmail returns workspace id matching user email.
func GetWorkspaceByUserEmail(s *State, email string) (string, *Workspace, error) {
	for id, ws := range s.Workspaces {
		if ws != nil && ws.UserEmail == email {
			return id, ws, nil
		}
	}
	return "", nil, ErrNotFound
}

// FindWorkspaceByFields returns a workspace matching all identity fields.
func FindWorkspaceByFields(s *State, userName, userEmail, signingKey, editor, gpgProgram string) (string, *Workspace, bool) {
	for id, ws := range s.Workspaces {
		if ws != nil && ws.UserName == userName && ws.UserEmail == userEmail &&
			ws.SigningKey == signingKey && ws.Editor == editor && ws.GPGProgram == gpgProgram {
			return id, ws, true
		}
	}
	return "", nil, false
}

// CreateWorkspaceInput holds fields for a new workspace.
type CreateWorkspaceInput struct {
	Name       string
	UserName   string
	UserEmail  string
	SigningKey string
	Editor     string
	GPGProgram string
}

// CreateWorkspace adds a workspace and returns its id.
func CreateWorkspace(s *State, in CreateWorkspaceInput) (string, error) {
	if in.Name == "" || in.UserName == "" || in.UserEmail == "" {
		return "", fmt.Errorf("name, user_name, and user_email are required")
	}
	for _, ws := range s.Workspaces {
		if ws != nil && ws.Name == in.Name {
			return "", fmt.Errorf("workspace %q already exists", in.Name)
		}
	}
	id, err := uuidv7.New()
	if err != nil {
		return "", err
	}
	s.Workspaces[id] = &Workspace{
		Name:        in.Name,
		UserName:    in.UserName,
		UserEmail:   in.UserEmail,
		SigningKey:  in.SigningKey,
		Editor:      in.Editor,
		GPGProgram:  in.GPGProgram,
		LinkedRepos: []string{},
	}
	return id, nil
}

// UpdateWorkspaceInput holds optional workspace field updates.
type UpdateWorkspaceInput struct {
	UserName   *string
	UserEmail  *string
	SigningKey *string
	Editor     *string
	GPGProgram *string
}

// UpdateWorkspace updates workspace fields by id.
func UpdateWorkspace(s *State, id string, in UpdateWorkspaceInput) error {
	ws, err := GetWorkspace(s, id)
	if err != nil {
		return err
	}
	if in.UserName != nil {
		ws.UserName = *in.UserName
	}
	if in.UserEmail != nil {
		ws.UserEmail = *in.UserEmail
	}
	if in.SigningKey != nil {
		ws.SigningKey = *in.SigningKey
	}
	if in.Editor != nil {
		ws.Editor = *in.Editor
	}
	if in.GPGProgram != nil {
		ws.GPGProgram = *in.GPGProgram
	}
	return nil
}

// DeleteWorkspace removes a workspace when it has no linked repositories.
func DeleteWorkspace(s *State, id string) error {
	ws, err := GetWorkspace(s, id)
	if err != nil {
		return err
	}
	if len(ws.LinkedRepos) > 0 {
		names := linkedRepoNames(s, ws.LinkedRepos)
		return fmt.Errorf("workspace %q is linked to %d repository (-ies): %s.\nRun `git elegant repo configure <other>` in each (or `git elegant repo sync`) to relink before deleting",
			ws.Name, len(ws.LinkedRepos), strings.Join(names, ", "))
	}
	delete(s.Workspaces, id)
	return nil
}

func linkedRepoNames(s *State, ids []string) []string {
	var names []string
	for _, id := range ids {
		if r, err := GetRepo(s, id); err == nil && r != nil {
			names = append(names, r.Name)
		} else {
			names = append(names, id)
		}
	}
	return names
}
