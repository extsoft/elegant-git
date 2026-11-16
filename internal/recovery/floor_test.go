package recovery

import "testing"

func TestParseGitVersionAcceptsBuildLabel(t *testing.T) {
	got, err := ParseGitVersion("git version 2.54.0 (Apple Git-157)")
	if err != nil {
		t.Fatal(err)
	}
	if got.major != 2 || got.minor != 54 || got.patch != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestVersionFloorRejectsGit212(t *testing.T) {
	if err := versionAtLeast("git version 2.12.9", MinGitVersion); err == nil {
		t.Fatal("expected refusal")
	}
}

func TestVersionFloorAccepts213(t *testing.T) {
	if err := versionAtLeast("git version 2.13.0", MinGitVersion); err != nil {
		t.Fatal(err)
	}
}
