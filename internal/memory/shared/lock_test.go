package shared

import (
	"path/filepath"
	"testing"
)

func TestWithLockRunsFn(t *testing.T) {
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(t.TempDir(), "state.json"))
	n := 0
	if err := WithLock(func() error {
		n++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
}
