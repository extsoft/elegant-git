package git

import (
	"testing"
)

func TestConfigLocalUnsetLogsWhenPresent(t *testing.T) {
	m := NewMemoryRunner()
	m.Repo.LocalConfig["elegant-git.default-branch"] = "main"
	Use(m)
	if err := ConfigLocalUnset("elegant-git.default-branch"); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Repo.LocalConfig["elegant-git.default-branch"]; ok {
		t.Fatal("key should be removed")
	}
	found := false
	for _, c := range m.Calls {
		if len(c.Args) >= 4 && c.Args[0] == "config" && c.Args[2] == "--unset" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected logged unset call")
	}
}
