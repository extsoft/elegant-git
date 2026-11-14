package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	resolvedExecutable string
	resolveOnce        sync.Once
)

// Executable returns the git binary to invoke.
// On darwin, when PATH resolves git to /usr/bin/git, it returns <exec-path>/git
// when that file exists and is executable. Every other case returns "git".
func Executable() string {
	resolveOnce.Do(func() {
		resolvedExecutable = resolveExecutable(runtime.GOOS, exec.LookPath, gitExecPath, executableFile)
	})
	return resolvedExecutable
}

func resolveExecutable(goos string, lookPath func(string) (string, error), execPath func(string) (string, error), candidateOK func(string) bool) string {
	if goos != "darwin" {
		return "git"
	}
	found, err := lookPath("git")
	if err != nil || found != "/usr/bin/git" {
		return "git"
	}
	dir, err := execPath("/usr/bin/git")
	if err != nil {
		return "git"
	}
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "git"
	}
	candidate := filepath.Join(dir, "git")
	if !candidateOK(candidate) {
		return "git"
	}
	return candidate
}

func gitExecPath(bin string) (string, error) {
	out, err := exec.Command(bin, "--exec-path").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func executableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	return info.Mode()&0o111 != 0
}
