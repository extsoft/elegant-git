// Package legacy maps deprecated flat command names to new object-first paths.
package legacy

import "github.com/bees-hive/elegant-git/internal/cmdid"

// LegacyToPath maps legacy name -> new CLI args (without "elegant").
var LegacyToPath = map[string][]string{
	"acquire-git":        {"git", "configure"},
	"acquire-repository": {"repo", "configure"},
	"clone-repository":   {"repo", "clone"},
	"init-repository":    {"repo", "init"},
	"prune-repository":   {"repo", "prune"},
	"show-workflows":     {"hook", "list"},
	"make-workflow":      {"hook", "new"},
	"polish-workflow":    {"hook", "edit"},
	"start-work":         {"work", "start"},
	"save-work":          {"work", "save"},
	"amend-work":         {"work", "amend"},
	"show-work":          {"work", "list"},
	"polish-work":        {"work", "polish"},
	"actualize-work":     {"work", "sync"},
	"deliver-work":       {"work", "push"},
	"obtain-work":        {"work", "track"},
	"accept-work":        {"work", "accept"},
	"release-work":       {"release", "new"},
	"show-release-notes": {"release", "notes"},
}

// LegacyToID maps legacy flat name -> canonical id.
var LegacyToID = map[string]cmdid.ID{
	"acquire-git":        {Command: "git", Action: "configure"},
	"acquire-repository": {Command: "repo", Action: "configure"},
	"clone-repository":   {Command: "repo", Action: "clone"},
	"init-repository":    {Command: "repo", Action: "init"},
	"prune-repository":   {Command: "repo", Action: "prune"},
	"show-workflows":     {Command: "hook", Action: "list"},
	"make-workflow":      {Command: "hook", Action: "new"},
	"polish-workflow":    {Command: "hook", Action: "edit"},
	"start-work":         {Command: "work", Action: "start"},
	"save-work":          {Command: "work", Action: "save"},
	"amend-work":         {Command: "work", Action: "amend"},
	"show-work":          {Command: "work", Action: "list"},
	"polish-work":        {Command: "work", Action: "polish"},
	"actualize-work":     {Command: "work", Action: "sync"},
	"deliver-work":       {Command: "work", Action: "push"},
	"obtain-work":        {Command: "work", Action: "track"},
	"accept-work":        {Command: "work", Action: "accept"},
	"release-work":       {Command: "release", Action: "new"},
	"show-release-notes": {Command: "release", Action: "notes"},
}

// IDFromLegacy returns the canonical id for a legacy flat name.
func IDFromLegacy(legacy string) (cmdid.ID, bool) {
	id, ok := LegacyToID[legacy]
	return id, ok
}

// ParseID parses a legacy name or dotted canonical id.
func ParseID(s string) (cmdid.ID, bool) {
	if id, ok := LegacyToID[s]; ok {
		return id, true
	}
	return cmdid.ParseDotted(s)
}

// IDToLegacy returns the legacy flat name for a canonical id, if any.
func IDToLegacy(id cmdid.ID) (string, bool) {
	for legacy, cid := range LegacyToID {
		if cid == id {
			return legacy, true
		}
	}
	return "", false
}

// AllIDs returns every canonical command id.
func AllIDs() []cmdid.ID {
	ids := make([]cmdid.ID, 0, len(LegacyToID))
	seen := map[string]bool{}
	for _, id := range LegacyToID {
		k := id.String()
		if seen[k] {
			continue
		}
		seen[k] = true
		ids = append(ids, id)
	}
	return ids
}

// LegacyNames returns all legacy command names in stable order.
func LegacyNames() []string {
	return []string{
		"accept-work", "acquire-git", "acquire-repository", "actualize-work",
		"amend-work", "clone-repository", "deliver-work", "init-repository",
		"make-workflow", "obtain-work", "polish-work", "polish-workflow",
		"prune-repository", "release-work", "save-work", "show-commands",
		"show-release-notes", "show-work", "show-workflows", "start-work",
	}
}

// ReplacementCommand returns the canonical command path for a legacy name (e.g. "work start").
func ReplacementCommand(legacy string) string {
	path, ok := LegacyToPath[legacy]
	if !ok {
		return ""
	}
	return joinArgs(path)
}

// AliasValue returns the git alias value for a legacy name (elegant <new path>).
func AliasValue(legacy string) string {
	path, ok := LegacyToPath[legacy]
	if !ok {
		return ""
	}
	return "elegant " + joinArgs(path)
}

func joinArgs(parts []string) string {
	s := parts[0]
	for _, p := range parts[1:] {
		s += " " + p
	}
	return s
}
