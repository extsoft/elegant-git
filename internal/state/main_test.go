package state

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

const hermeticGitConfig = `[user]
	name = Test
	email = test@example.com
[commit]
	gpgsign = false
[maintenance]
	auto = false
[gc]
	auto = 0
`

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "eg-gitconfig-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "git config dir: %v\n", err)
		os.Exit(1)
	}
	cfg := filepath.Join(dir, "gitconfig")
	if err := os.WriteFile(cfg, []byte(hermeticGitConfig), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "git config: %v\n", err)
		os.RemoveAll(dir)
		os.Exit(1)
	}
	os.Setenv("GIT_CONFIG_GLOBAL", cfg)
	os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
