package shared

import (
	"fmt"
	"sort"
	"strings"

	"github.com/extsoft/elegant-git/internal/uuidv7"
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
	Namespaces []string
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
		Namespaces:  append([]string(nil), in.Namespaces...),
		LinkedRepos: []string{},
	}
	return id, nil
}

// AddWorkspaceNamespace appends a namespace to a workspace if not already present.
// Returns whether the list changed.
func AddWorkspaceNamespace(s *State, id, namespace string) (bool, error) {
	ws, err := GetWorkspace(s, id)
	if err != nil {
		return false, err
	}
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return false, nil
	}
	for _, n := range ws.Namespaces {
		if n == namespace {
			return false, nil
		}
	}
	ws.Namespaces = append(ws.Namespaces, namespace)
	return true, nil
}

// FindWorkspacesByNamespace returns workspace ids whose Namespaces contain
// namespace, ordered by workspace display name.
func FindWorkspacesByNamespace(s *State, namespace string) []string {
	if s == nil || namespace == "" {
		return nil
	}
	type hit struct {
		id   string
		name string
	}
	var hits []hit
	for id, ws := range s.Workspaces {
		if ws == nil {
			continue
		}
		for _, n := range ws.Namespaces {
			if n == namespace {
				hits = append(hits, hit{id: id, name: ws.Name})
				break
			}
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].name < hits[j].name })
	out := make([]string, len(hits))
	for i, h := range hits {
		out[i] = h.id
	}
	return out
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

// DeleteWorkspace removes a workspace and clears workspace_id on linked
// repository registry entries. Repository working copies and git config are
// left untouched.
func DeleteWorkspace(s *State, id string) error {
	ws, err := GetWorkspace(s, id)
	if err != nil {
		return err
	}
	linked := append([]string(nil), ws.LinkedRepos...)
	for _, repoID := range linked {
		repo, err := GetRepo(s, repoID)
		if err != nil {
			continue
		}
		repo.WorkspaceID = ""
	}
	delete(s.Workspaces, id)
	return nil
}
