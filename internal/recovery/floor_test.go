package recovery

import (
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
)

func TestCheckFloorUsesSharedPolicy(t *testing.T) {
	m := git.NewMemoryRunner()
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	m.Outputs["version"] = "git version 2.39.0"
	if err := CheckFloor(); err == nil || !strings.Contains(err.Error(), MinGitVersion) {
		t.Fatalf("below floor: %v", err)
	}
	m.Outputs["version"] = "git version 2.13.0"
	if err := CheckFloor(); err == nil {
		t.Fatal("2.13 is not the product floor")
	}
	m.Outputs["version"] = "git version 2.40.0-rc1"
	if err := CheckFloor(); err == nil || !strings.Contains(err.Error(), "not a stable release") {
		t.Fatalf("prerelease: %v", err)
	}
	for _, ver := range []string{
		"git version 2.40.0",
		"git version 2.54.0 (Apple Git-157)",
		"git version 2.40.0.windows.1",
		"git version 3.1.0",
	} {
		m.Outputs["version"] = ver
		if err := CheckFloor(); err != nil {
			t.Fatalf("%s: %v", ver, err)
		}
	}
	delete(m.Outputs, "version")
	m.FailOn["version"] = errString("missing")
	if err := CheckFloor(); err == nil || !strings.Contains(err.Error(), "git version failed") {
		t.Fatalf("missing: %v", err)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
