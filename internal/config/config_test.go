package config

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/version"
)

func TestNeedsLocalGitInstall(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	m := git.NewMemoryRunner()
	git.Use(m)
	if !NeedsLocalGitInstall() {
		t.Fatal("expected true without global acquired")
	}
	s, _ := shared.Load()
	shared.SetAcquired(s, "1.0.0")
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	if NeedsLocalGitInstall() {
		t.Fatal("expected false when global acquired in shared memory")
	}
}

func TestMarkAcquiredWritesSharedMemory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	m := git.NewMemoryRunner()
	m.GlobalConfig[AcquiredKey] = AcquiredValueLegacy
	git.Use(m)
	if err := MarkAcquired("--global"); err != nil {
		t.Fatal(err)
	}
	loaded, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	if shared.Acquired(loaded) != version.Version {
		t.Fatalf("got %q", shared.Acquired(loaded))
	}
	assertConfigCall(t, m.Calls, "--global", "--unset", AcquiredKey)
}

func TestImportAcquiredToSharedMemory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	m := git.NewMemoryRunner()
	m.GlobalConfig[AcquiredKey] = AcquiredValueLegacy
	git.Use(m)
	if err := importAcquiredToSharedMemory("--global"); err != nil {
		t.Fatal(err)
	}
	loaded, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	if shared.Acquired(loaded) != version.Version {
		t.Fatalf("got %q", shared.Acquired(loaded))
	}
	assertConfigCall(t, m.Calls, "--global", "--unset", AcquiredKey)
}

func TestRemoveObsoleteAcquiredUnsetsLocalKey(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["config --local --get-regexp "+AcquiredKey] = AcquiredKey + " true"
	git.Use(m)
	if err := RemoveObsoleteAcquired("--local", false); err != nil {
		t.Fatal(err)
	}
	assertConfigCall(t, m.Calls, "--local", "--unset", AcquiredKey)
}

func TestRemoveObsoleteAcquiredNoOpWhenUnset(t *testing.T) {
	m := git.NewMemoryRunner()
	git.Use(m)
	if err := RemoveObsoleteAcquired("--local", false); err != nil {
		t.Fatal(err)
	}
	for _, c := range m.Calls {
		if len(c.Args) >= 3 && c.Args[2] == "--unset" {
			t.Fatalf("unexpected unset: %+v", c.Args)
		}
	}
}

func TestCleanupRedundantLocalInstallRemovesAliases(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["config --local --get-regexp "+AcquiredKey] = AcquiredKey + " true"
	m.Outputs["config --local --get-regexp ^alias\\."] = "alias.start-work elegant work start"
	git.Use(m)
	if err := CleanupRedundantLocalInstall(false); err != nil {
		t.Fatal(err)
	}
	assertConfigCall(t, m.Calls, "--local", "--unset", "alias.start-work")
}

func TestObsoleteConfigurationsRemovingAlsoRemovesAliases(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["config --local --get-regexp "+AcquiredKey] = AcquiredKey + " true"
	m.Outputs["config --local --get-regexp ^alias\\."] = "alias.start-work elegant work start"
	git.Use(m)
	if err := ObsoleteConfigurationsRemoving("--local"); err != nil {
		t.Fatal(err)
	}
	assertConfigCall(t, m.Calls, "--local", "--unset", "alias.start-work")
}

func TestAliasesRemovingOnlyElegantAliases(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["config --local --get-regexp ^alias\\."] = "alias.start-work elegant work start\nalias.st status"
	git.Use(m)
	if err := AliasesRemoving("--local", false); err != nil {
		t.Fatal(err)
	}
	unsetStart := false
	for _, c := range m.Calls {
		if len(c.Args) >= 4 && c.Args[0] == "config" && c.Args[2] == "--unset" {
			if strings.HasSuffix(c.Args[3], "start-work") {
				unsetStart = true
			}
			if strings.HasSuffix(c.Args[3], "st") {
				t.Fatal("non-elegant alias should not be removed")
			}
		}
	}
	if !unsetStart {
		t.Fatal("expected start-work alias unset")
	}
}

func TestAliasesRemovingKeepsElegantAlias(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["config --local --get-regexp ^alias\\."] = strings.Join([]string{
		"alias.elegant !eg",
		"alias.start-work elegant work start",
		"alias.save-work !eg work save",
		"alias.st status",
	}, "\n")
	git.Use(m)
	if err := AliasesRemoving("--local", false); err != nil {
		t.Fatal(err)
	}
	unset := map[string]bool{}
	for _, c := range m.Calls {
		if len(c.Args) >= 4 && c.Args[0] == "config" && c.Args[2] == "--unset" {
			unset[c.Args[3]] = true
		}
	}
	if !unset["alias.start-work"] {
		t.Fatal("expected start-work (elegant …) unset")
	}
	if !unset["alias.save-work"] {
		t.Fatal("expected save-work (!eg …) unset")
	}
	if unset[ElegantAliasKey] {
		t.Fatal("alias.elegant must survive removal")
	}
	if unset["alias.st"] {
		t.Fatal("non-elegant alias should not be removed")
	}
}

func TestAliasesConfigurationWritesElegantAndFlat(t *testing.T) {
	m := git.NewMemoryRunner()
	git.Use(m)
	if err := AliasesConfiguration("--global"); err != nil {
		t.Fatal(err)
	}
	if m.GlobalConfig[ElegantAliasKey] != ElegantAliasValue {
		t.Fatalf("alias.elegant = %q", m.GlobalConfig[ElegantAliasKey])
	}
	if m.GlobalConfig["alias.start-work"] != "!eg work start" {
		t.Fatalf("alias.start-work = %q", m.GlobalConfig["alias.start-work"])
	}
}

func TestAliasesRemovingDoesNotRecordDEP015(t *testing.T) {
	deprecation.Reset()
	m := git.NewMemoryRunner()
	m.Outputs["config --local --get-regexp ^alias\\."] = "alias.start-work elegant work start"
	git.Use(m)
	if err := AliasesRemoving("--local", false); err != nil {
		t.Fatal(err)
	}
	for _, e := range deprecation.Events() {
		if e.ID == deprecation.DEP015 {
			t.Fatal("configure/remove must not record DEP-015")
		}
	}
}

func TestIsRemovableAliasValue(t *testing.T) {
	if !IsRemovableAliasValue("elegant work start") {
		t.Fatal("stale elegant prefix")
	}
	if !IsRemovableAliasValue("!eg work save") {
		t.Fatal("!eg prefix")
	}
	if IsRemovableAliasValue(ElegantAliasValue) {
		t.Fatal("bare !eg must not be removable")
	}
	if IsRemovableAliasValue("status") {
		t.Fatal("unrelated alias")
	}
}

func assertConfigCall(t *testing.T, calls []git.Call, scope, verb, key string) {
	t.Helper()
	for _, c := range calls {
		if len(c.Args) >= 4 && c.Args[0] == "config" && c.Args[1] == scope && c.Args[2] == verb && c.Args[3] == key {
			return
		}
	}
	t.Fatalf("missing config %s %s %s in %+v", scope, verb, key, calls)
}
