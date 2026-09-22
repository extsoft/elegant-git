package git

import "testing"

func TestValidateBranchName(t *testing.T) {
	valid := []string{"main", "feature/foo", "datadog-alerts", "OPS-2162"}
	for _, name := range valid {
		if err := validateBranchName(name); err != nil {
			t.Errorf("%q: %v", name, err)
		}
	}
	invalid := []string{"", ".hidden", "bad name", "foo..bar", "ends/", "trail.", "has@{", "a.lock"}
	for _, name := range invalid {
		if err := validateBranchName(name); err == nil {
			t.Errorf("%q: expected error", name)
		}
	}
}

func TestCheckBranchNameMemoryRunner(t *testing.T) {
	m := NewMemoryRunner()
	Use(m)
	t.Cleanup(func() { Use(RealRunner{}) })
	if err := CheckBranchName("datadog-alerts"); err != nil {
		t.Fatal(err)
	}
	if err := CheckBranchName("bad name"); err == nil {
		t.Fatal("expected error for space in branch name")
	}
}
