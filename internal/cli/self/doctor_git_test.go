package self

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/spf13/cobra"
)

func TestDoctorReportsUnsupportedGitWithoutRepair(t *testing.T) {
	git.Use(git.RealRunner{})
	git.SetVersionProbeForTest(func(string) (string, error) {
		return "git version 2.13.0", nil
	})
	t.Cleanup(func() {
		git.SetVersionProbeForTest(nil)
		git.Use(git.RealRunner{})
	})
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "doctor"}
	cmd.SetOut(&buf)
	err := doctorRun(cmd)
	var status cliruntime.ExitStatus
	if !errors.As(err, &status) || status.Code != 1 {
		t.Fatalf("err=%#v", err)
	}
	out := buf.String()
	if !strings.Contains(out, git.MinVersion) || !strings.Contains(out, "2.13.0") || !strings.Contains(out, "PATH") {
		t.Fatalf("out=%s", out)
	}
	if strings.Contains(out, "healthy") || strings.Contains(out, "Repaired") {
		t.Fatalf("out=%s", out)
	}
}

func TestDoctorReportsMissingGit(t *testing.T) {
	git.Use(git.RealRunner{})
	git.SetVersionProbeForTest(func(string) (string, error) {
		return "", errors.New("executable file not found")
	})
	t.Cleanup(func() {
		git.SetVersionProbeForTest(nil)
		git.Use(git.RealRunner{})
	})
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "doctor"}
	cmd.SetOut(&buf)
	err := doctorRun(cmd)
	var status cliruntime.ExitStatus
	if !errors.As(err, &status) || status.Code != 1 {
		t.Fatalf("err=%#v", err)
	}
	if !strings.Contains(buf.String(), "git version failed") || !strings.Contains(buf.String(), git.MinVersion) {
		t.Fatalf("out=%s", buf.String())
	}
}
