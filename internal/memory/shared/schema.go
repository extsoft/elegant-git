package shared

import "github.com/extsoft/elegant-git/internal/deprecation"

// stateWire accepts both v1 (profiles/profile_id) and v2 (workspaces/workspace_id) keys.
type stateWire struct {
	SchemaVersion   int                   `json:"schema_version"`
	AcquiredVersion string                `json:"acquired_version,omitempty"`
	Workspaces      map[string]*Workspace `json:"workspaces"`
	Profiles        map[string]*Workspace `json:"profiles"` // v1
	Repositories    map[string]*repoWire  `json:"repositories"`
}

type repoWire struct {
	Name        string   `json:"name"`
	WorkspaceID string   `json:"workspace_id"`
	ProfileID   string   `json:"profile_id"` // v1
	CurrentPath string   `json:"current_path"`
	PathHistory []string `json:"path_history,omitempty"`
	OriginURL   string   `json:"origin_url,omitempty"`
}

var lastLoadHadLegacyKeys bool

// LastLoadHadLegacyKeys reports whether the most recent Load saw v1 keys.
func LastLoadHadLegacyKeys() bool {
	return lastLoadHadLegacyKeys
}

func decodeState(w stateWire) (*State, bool) {
	hadLegacy := w.Profiles != nil
	workspaces := w.Workspaces
	if len(workspaces) == 0 && len(w.Profiles) > 0 {
		workspaces = w.Profiles
	}
	if workspaces == nil {
		workspaces = map[string]*Workspace{}
	}
	repos := map[string]*Repository{}
	for id, rw := range w.Repositories {
		if rw == nil {
			continue
		}
		if rw.ProfileID != "" {
			hadLegacy = true
		}
		wsID := rw.WorkspaceID
		if wsID == "" {
			wsID = rw.ProfileID
		}
		repos[id] = &Repository{
			Name:        rw.Name,
			WorkspaceID: wsID,
			CurrentPath: rw.CurrentPath,
			PathHistory: rw.PathHistory,
			OriginURL:   rw.OriginURL,
		}
	}
	s := &State{
		SchemaVersion:   SchemaVersion,
		AcquiredVersion: w.AcquiredVersion,
		Workspaces:      workspaces,
		Repositories:    repos,
	}
	return s, hadLegacy
}

func recordLegacyMemoryKeys() {
	deprecation.Record(
		deprecation.DEP012,
		"shared/per-repo memory keys: profiles, profile_id",
		"workspaces, workspace_id",
		"git elegant repo migrate",
	)
}
