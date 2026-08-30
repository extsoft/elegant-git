//go:build unix

package shared

import (
	"os"

	"golang.org/x/sys/unix"
)

func lockExclusive(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX)
}

func unlockExclusive(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}
