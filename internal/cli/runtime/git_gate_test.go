package runtime

import (
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	"github.com/spf13/cobra"
)

func TestRunBoundRefusesUnsupportedGitBeforeHandler(t *testing.T) {
	git.SetVersionProbeForTest(func(string) (string, error) {
		return "git version 2.39.0", nil
	})
	t.Cleanup(func() { git.SetVersionProbeForTest(nil) })
	called := false
	sub := &cobra.Command{Use: "start", RunE: func(*cobra.Command, []string) error {
		called = true
		return nil
	}}
	err := RunBound(&cobra.Command{Use: "work"}, sub, nil)
	if err == nil || called || !strings.Contains(err.Error(), git.MinVersion) {
		t.Fatalf("called=%v err=%v", called, err)
	}
}

func TestRunBoundAllowsSupportedGit(t *testing.T) {
	git.SetVersionProbeForTest(func(string) (string, error) {
		return "git version 2.40.0", nil
	})
	t.Cleanup(func() { git.SetVersionProbeForTest(nil) })
	called := false
	sub := &cobra.Command{Use: "start", RunE: func(*cobra.Command, []string) error {
		called = true
		return nil
	}}
	if err := RunBound(&cobra.Command{Use: "work"}, sub, nil); err != nil || !called {
		t.Fatalf("called=%v err=%v", called, err)
	}
}
