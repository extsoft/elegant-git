//go:build unix

package recovery

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestRecoveryLockBlocksAnotherProcess(t *testing.T) {
	dir := t.TempDir()
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- WithLock(dir, func() error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started
	cmd := exec.Command(os.Args[0], "-test.run=^TestRecoveryLockChildProbe$", "-test.v")
	cmd.Env = append(os.Environ(), "EG_LOCK_CHILD=1", "EG_LOCK_DIR="+dir)
	out, err := cmd.CombinedOutput()
	close(release)
	if waitErr := <-done; waitErr != nil {
		t.Fatal(waitErr)
	}
	if err != nil {
		t.Fatalf("child acquired the lock: %v\n%s", err, out)
	}
}

func TestRecoveryLockChildProbe(t *testing.T) {
	if os.Getenv("EG_LOCK_CHILD") != "1" {
		t.Skip()
	}
	f, err := os.Open(filepath.Join(os.Getenv("EG_LOCK_DIR"), "elegant-git", "recovery.lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err == nil {
		t.Fatal("lock was not held")
	}
}
