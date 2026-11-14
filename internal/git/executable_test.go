package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveExecutable(t *testing.T) {
	const xcode = "/Applications/Xcode.app/Contents/Developer/usr/libexec/git-core"
	shim := func(string) (string, error) { return "/usr/bin/git", nil }
	tests := []struct {
		name string
		goos string
		look func(string) (string, error)
		exec func(string) (string, error)
		ok   func(string) bool
		want string
	}{
		{
			name: "darwin shim",
			goos: "darwin",
			look: func(name string) (string, error) {
				if name != "git" {
					return "", errors.New("LookPath " + name)
				}
				return "/usr/bin/git", nil
			},
			exec: func(bin string) (string, error) {
				if bin != "/usr/bin/git" {
					return "", errors.New("unexpected bin " + bin)
				}
				return xcode + "\n", nil
			},
			ok: func(path string) bool {
				return path == filepath.Join(xcode, "git")
			},
			want: filepath.Join(xcode, "git"),
		},
		{
			name: "missing candidate",
			goos: "darwin",
			look: shim,
			exec: func(string) (string, error) { return xcode, nil },
			ok:   func(string) bool { return false },
			want: "git",
		},
		{
			name: "exec-path fails",
			goos: "darwin",
			look: shim,
			exec: func(string) (string, error) { return "", errors.New("exec-path failed") },
			want: "git",
		},
		{
			name: "empty exec-path",
			goos: "darwin",
			look: shim,
			exec: func(string) (string, error) { return " \n", nil },
			want: "git",
		},
		{
			name: "linux",
			goos: "linux",
			want: "git",
		},
		{
			name: "other path git",
			goos: "darwin",
			look: func(string) (string, error) { return "/opt/homebrew/bin/git", nil },
			want: "git",
		},
		{
			name: "lookpath error",
			goos: "darwin",
			look: func(string) (string, error) { return "", errors.New("not found") },
			want: "git",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			look := tc.look
			if look == nil {
				look = func(string) (string, error) {
					t.Fatal("LookPath called")
					return "", nil
				}
			}
			execPath := tc.exec
			if execPath == nil {
				execPath = func(string) (string, error) {
					t.Fatal("exec-path called")
					return "", nil
				}
			}
			ok := tc.ok
			if ok == nil {
				ok = func(string) bool {
					t.Fatal("candidate checked")
					return false
				}
			}
			got := resolveExecutable(tc.goos, look, execPath, ok)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestExecutableFile(t *testing.T) {
	dir := t.TempDir()
	if executableFile(filepath.Join(dir, "missing")) {
		t.Fatal("missing file")
	}
	if executableFile(dir) {
		t.Fatal("directory")
	}
	file := filepath.Join(dir, "git")
	if err := os.WriteFile(file, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if executableFile(file) {
		t.Fatal("mode 0644")
	}
	if err := os.Chmod(file, 0o755); err != nil {
		t.Fatal(err)
	}
	if !executableFile(file) {
		t.Fatal("mode 0755")
	}
}

func TestExecutableResolvesShimOnDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("shim resolution is darwin-only")
	}
	found, err := exec.LookPath("git")
	if err != nil || found != "/usr/bin/git" {
		t.Skip("PATH git is not /usr/bin/git")
	}
	out, err := exec.Command("/usr/bin/git", "--exec-path").Output()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(strings.TrimSpace(string(out)), "git")
	got := Executable()
	if got != want {
		t.Fatalf("Executable() = %q, want %q", got, want)
	}
	if !executableFile(got) {
		t.Fatalf("%s is not executable", got)
	}
}

func TestExecutableFallsBackWhenGitExecPathHasNoGit(t *testing.T) {
	if os.Getenv("EG_RESOLVE_GIT_EXEC_PATH") == "1" {
		if got := Executable(); got != "git" {
			t.Fatalf("Executable() = %q", got)
		}
		return
	}
	if runtime.GOOS != "darwin" {
		t.Skip("GIT_EXEC_PATH fallback is exercised on darwin")
	}
	found, err := exec.LookPath("git")
	if err != nil || found != "/usr/bin/git" {
		t.Skip("PATH git is not /usr/bin/git")
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestExecutableFallsBackWhenGitExecPathHasNoGit$")
	env := make([]string, 0, len(os.Environ())+2)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GIT_EXEC_PATH=") {
			continue
		}
		env = append(env, e)
	}
	cmd.Env = append(env, "EG_RESOLVE_GIT_EXEC_PATH=1", "GIT_EXEC_PATH="+t.TempDir())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fallback: %v\n%s", err, out)
	}
}
