package shared

import (
	"os"
	"path/filepath"
)

// WithLock serializes fn against other elegant-git processes using a sibling
// lock file next to shared memory. Same-process calls from two goroutines are
// also serialized via mu after the file lock is held.
func WithLock(fn func() error) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(p+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := lockExclusive(f); err != nil {
		return err
	}
	defer unlockExclusive(f)
	return fn()
}
