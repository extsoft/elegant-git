package recovery

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/extsoft/elegant-git/internal/git"
)

// MinGitVersion is the oldest Git release that can prepare a backup.
// Git 2.13 added `git stash push`. Preparation also uses
// `rev-parse --git-common-dir` and `update-ref -d <ref> <oldvalue>`.
const MinGitVersion = "2.13.0"

// CheckFloor refuses backup preparation on unsupported Git or operating systems
// before any stash or journal write.
func CheckFloor() error {
	switch runtime.GOOS {
	case "linux", "darwin", "windows":
	default:
		return fmt.Errorf("backup preparation is not supported on %s", runtime.GOOS)
	}
	out, err := git.Output("version")
	if err != nil {
		return fmt.Errorf("backup preparation cannot read the git version: %w", err)
	}
	if err := versionAtLeast(out, MinGitVersion); err != nil {
		return err
	}
	return nil
}

func versionAtLeast(got, min string) error {
	g, err := ParseGitVersion(got)
	if err != nil {
		return fmt.Errorf("backup preparation cannot read the git version: %w", err)
	}
	m, err := ParseGitVersion("git version " + min)
	if err != nil {
		return err
	}
	if compareVersion(g, m) < 0 {
		return fmt.Errorf("backup preparation requires Git %s or newer; this Git is %s", min, strings.TrimSpace(got))
	}
	return nil
}

type gitVersion struct {
	major, minor, patch int
}

// ParseGitVersion reads "git version X.Y.Z" and ignores a trailing build label.
func ParseGitVersion(s string) (gitVersion, error) {
	s = strings.TrimSpace(s)
	const prefix = "git version "
	if !strings.HasPrefix(s, prefix) {
		return gitVersion{}, fmt.Errorf("unrecognized git version %q", s)
	}
	fields := strings.Fields(strings.TrimPrefix(s, prefix))
	if len(fields) == 0 {
		return gitVersion{}, fmt.Errorf("unrecognized git version %q", s)
	}
	parts := strings.Split(fields[0], ".")
	if len(parts) < 2 {
		return gitVersion{}, fmt.Errorf("unrecognized git version %q", s)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return gitVersion{}, fmt.Errorf("unrecognized git version %q", s)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return gitVersion{}, fmt.Errorf("unrecognized git version %q", s)
	}
	patch := 0
	if len(parts) > 2 {
		patch, err = leadingInt(parts[2])
		if err != nil {
			return gitVersion{}, fmt.Errorf("unrecognized git version %q", s)
		}
	}
	return gitVersion{major: major, minor: minor, patch: patch}, nil
}

func leadingInt(s string) (int, error) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, fmt.Errorf("no number")
	}
	return strconv.Atoi(s[:i])
}

func compareVersion(a, b gitVersion) int {
	if a.major != b.major {
		return cmpInt(a.major, b.major)
	}
	if a.minor != b.minor {
		return cmpInt(a.minor, b.minor)
	}
	return cmpInt(a.patch, b.patch)
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
