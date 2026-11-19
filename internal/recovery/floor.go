package recovery

import (
	"fmt"
	"runtime"

	"github.com/extsoft/elegant-git/internal/git"
)

// MinGitVersion is the product Git floor. Backup preparation uses the same
// check as every other Git-backed command, and still refuses unsupported
// operating systems before any stash or journal write.
const MinGitVersion = git.MinVersion

// CheckFloor refuses backup preparation on unsupported Git or operating systems
// before any stash or journal write.
func CheckFloor() error {
	switch runtime.GOOS {
	case "linux", "darwin", "windows":
	default:
		return fmt.Errorf("backup preparation is not supported on %s", runtime.GOOS)
	}
	if err := git.Check(); err != nil {
		return fmt.Errorf("backup preparation requires Git %s or newer: %w", MinGitVersion, err)
	}
	return nil
}
