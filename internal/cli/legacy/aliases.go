// Package legacy re-exports command-name maps and hosts hidden Cobra shims.
package legacy

import (
	"github.com/extsoft/elegant-git/internal/cmdid"
	maplegacy "github.com/extsoft/elegant-git/internal/legacy"
)

var (
	LegacyToPath = maplegacy.LegacyToPath
	LegacyToID   = maplegacy.LegacyToID
)

func IDFromLegacy(name string) (cmdid.ID, bool) { return maplegacy.IDFromLegacy(name) }
func ParseID(s string) (cmdid.ID, bool)         { return maplegacy.ParseID(s) }
func IDToLegacy(id cmdid.ID) (string, bool)     { return maplegacy.IDToLegacy(id) }
func AllIDs() []cmdid.ID                        { return maplegacy.AllIDs() }
func LegacyNames() []string                     { return maplegacy.LegacyNames() }
func ReplacementCommand(name string) string     { return maplegacy.ReplacementCommand(name) }
func AliasValue(name string) string             { return maplegacy.AliasValue(name) }
func NewArgPath(name string) string             { return maplegacy.NewArgPath(name) }
