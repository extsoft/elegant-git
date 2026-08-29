package git

import (
	"path/filepath"
	"testing"

	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/deprecation"
	rungit "github.com/extsoft/elegant-git/internal/git"
)

func TestMigrateGlobalRewritesStaleAliases(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	deprecation.Reset()
	m := rungit.NewMemoryRunner()
	m.GlobalConfig["alias.save-work"] = "elegant work save"
	m.GlobalConfig["alias.start-work"] = "elegant start-work"
	rungit.Use(m)

	if err := migrateGlobal(false); err != nil {
		t.Fatal(err)
	}
	if m.GlobalConfig[config.ElegantAliasKey] != config.ElegantAliasValue {
		t.Fatalf("alias.elegant = %q", m.GlobalConfig[config.ElegantAliasKey])
	}
	if m.GlobalConfig["alias.save-work"] != "!eg work save" {
		t.Fatalf("alias.save-work = %q", m.GlobalConfig["alias.save-work"])
	}
	if m.GlobalConfig["alias.start-work"] != "!eg work start" {
		t.Fatalf("alias.start-work = %q", m.GlobalConfig["alias.start-work"])
	}
	found := false
	for _, e := range deprecation.Events() {
		if e.ID == deprecation.DEP015 {
			found = true
		}
	}
	if !found {
		t.Fatal("expected DEP-015 for stale elegant alias")
	}
}
