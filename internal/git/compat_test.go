package git

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseVersionPolicy(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
		stable  bool
		major   int
		minor   int
		patch   int
	}{
		{in: "git version 2.40.0", stable: true, major: 2, minor: 40},
		{in: "git version 2.40.1", stable: true, major: 2, minor: 40, patch: 1},
		{in: "git version 2.56.1", stable: true, major: 2, minor: 56, patch: 1},
		{in: "git version 3.0.0", stable: true, major: 3},
		{in: "git version 2.54.0 (Apple Git-157)", stable: true, major: 2, minor: 54},
		{in: "git version 2.40.0.windows.1", stable: true, major: 2, minor: 40},
		{in: "git version 2.41.0.windows.2", stable: true, major: 2, minor: 41},
		{in: "git version 2.39.0", stable: true, major: 2, minor: 39},
		{in: "git version 2.13.0", stable: true, major: 2, minor: 13},
		{in: "git version 2.12.9", stable: true, major: 2, minor: 12, patch: 9},
		{in: "git version 2.40.0-rc1", stable: false, major: 2, minor: 40},
		{in: "git version 2.41.0-rc0", stable: false, major: 2, minor: 41},
		{in: "git version 2.40.0rc1", wantErr: true},
		{in: "git version 2.40.0.rc1", wantErr: true},
		{in: "git version 2.40.0.windows", wantErr: true},
		{in: "git version 2.40", wantErr: true},
		{in: "git version", wantErr: true},
		{in: "2.40.0", wantErr: true},
		{in: "git version 2.40.0 junk", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			token, _, err := versionToken(firstVersionLine(tc.in))
			if tc.wantErr {
				if err == nil {
					if _, perr := parseToken(token); perr == nil {
						t.Fatal("expected refusal")
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := parseToken(token)
			if err != nil {
				t.Fatal(err)
			}
			if got.stable != tc.stable || got.major != tc.major || got.minor != tc.minor || got.patch != tc.patch {
				t.Fatalf("%+v", got)
			}
		})
	}
}

func TestCheckRefusesBelowFloorAndPrerelease(t *testing.T) {
	for _, ver := range []string{
		"2.12.9", "2.13.0", "2.23.0", "2.29.0", "2.30.0", "2.34.0", "2.38.0", "2.39.0", "2.40.0-rc1", "2.41.0-rc0",
	} {
		err := decide("git", "git version "+ver, nil)
		exe := selectedExecutable("git")
		if err == nil || !strings.Contains(err.Error(), MinVersion) || !strings.Contains(err.Error(), ver) || !strings.Contains(err.Error(), "("+exe+")") || !strings.Contains(err.Error(), "PATH") {
			t.Fatalf("%s: %v", ver, err)
		}
	}
	for _, ver := range []string{"2.40.0", "2.40.1", "2.56.1", "3.0.0", "2.40.0.windows.1"} {
		if err := decide("/usr/bin/git", "git version "+ver, nil); err != nil {
			t.Fatalf("%s: %v", ver, err)
		}
	}
	if err := decide("/opt/git", "git version 2.54.0 (Apple Git-157)", nil); err != nil {
		t.Fatal(err)
	}
}

func TestCheckReportsMissingGit(t *testing.T) {
	err := decide("git", "", fmt.Errorf("executable file not found"))
	exe := selectedExecutable("git")
	if err == nil || !strings.Contains(err.Error(), "git version failed") || !strings.Contains(err.Error(), "("+exe+")") || !strings.Contains(err.Error(), MinVersion) {
		t.Fatal(err)
	}
	if strings.Count(err.Error(), "executable file not found") != 1 {
		t.Fatalf("cause repeated: %v", err)
	}
}

func TestVersionLineSkipsLeadingTrace(t *testing.T) {
	out := "10:00:00.000000 trace: built-in: git version\ngit version 2.54.0 (Apple Git-157)\n"
	if err := decide("/opt/git", out, nil); err != nil {
		t.Fatal(err)
	}
	if got := firstVersionLine(out); got != "git version 2.54.0 (Apple Git-157)" {
		t.Fatalf("line %q", got)
	}
}

func TestProbeReadsStdoutWhenTraceIsSet(t *testing.T) {
	t.Setenv("GIT_TRACE", "1")
	out, err := probeExecutable(Executable())
	if err != nil {
		t.Fatal(err)
	}
	if err := decide(Executable(), out, nil); err != nil {
		t.Fatalf("out %q: %v", out, err)
	}
}

func TestCheckCachesRealProbeOnly(t *testing.T) {
	seen := map[string]int{}
	SetVersionProbeForTest(func(exe string) (string, error) {
		seen[exe]++
		return "git version 2.40.0", nil
	})
	t.Cleanup(func() {
		SetVersionProbeForTest(nil)
		Use(RealRunner{})
	})
	Use(RealRunner{})
	if err := checkExe("/opt/a/git"); err != nil {
		t.Fatal(err)
	}
	if err := checkExe("/opt/a/git"); err != nil {
		t.Fatal(err)
	}
	if err := checkExe("/opt/b/git"); err != nil {
		t.Fatal(err)
	}
	if seen["/opt/a/git"] != 1 || seen["/opt/b/git"] != 1 {
		t.Fatalf("probes=%v", seen)
	}
	m := NewMemoryRunner()
	m.Outputs["version"] = "git version 2.13.0"
	Use(m)
	if err := Check(); err == nil {
		t.Fatal("injected runner used the real cache")
	}
}
