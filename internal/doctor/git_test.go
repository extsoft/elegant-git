package doctor

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
)

func TestCurrentRepoFindsLegacyHooks(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	if err := os.MkdirAll(filepath.Join(dir, ".git", ".workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".workflows", "start-work-ahead"), []byte("echo\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", ".workflows", "save-work-after"), []byte("echo\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(dir, ".git")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	findings, err := CurrentRepo(s, prompt.NewNonInteractive(), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	var personal, common bool
	for _, f := range findings {
		if strings.Contains(f.Problem, ".git/.workflows") {
			personal = true
			if f.Apply == nil {
				t.Fatal("personal hooks must be repairable")
			}
		}
		if strings.Contains(f.Problem, ".workflows/") && !strings.Contains(f.Problem, ".git/.workflows") {
			common = true
			if f.Apply == nil {
				t.Fatal("tracked hooks must be repairable")
			}
		}
	}
	if !personal || !common {
		t.Fatalf("personal=%v common=%v findings=%v", personal, common, problems(findings))
	}
}

func TestCurrentRepoFindsGitPrefixedHooks(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	commonDir := filepath.Join(dir, ".config", "elegant-git", "hooks")
	personalDir := filepath.Join(dir, ".git", ".config", "elegant-git", "hooks")
	if err := os.MkdirAll(commonDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(personalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(commonDir, "git-configure-ahead"), []byte("echo\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(personalDir, "git-doctor-after"), []byte("echo\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	m := git.NewMemoryRunner()
	m.Outputs["rev-parse --git-dir"] = filepath.Join(dir, ".git")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	findings, err := CurrentRepo(s, prompt.NewNonInteractive(), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	var personal, common bool
	for _, f := range findings {
		if strings.Contains(f.Problem, "personal hook files still use the git- command prefix") {
			personal = true
			if f.Apply == nil {
				t.Fatal("personal git- hooks must be repairable")
			}
		}
		if strings.Contains(f.Problem, "repo-tracked hook files still use the git- command prefix") {
			common = true
			if f.Apply == nil {
				t.Fatal("tracked git- hooks must be repairable")
			}
		}
	}
	if !personal || !common {
		t.Fatalf("personal=%v common=%v findings=%v", personal, common, problems(findings))
	}
}

func TestGitInstallRewritesDriftedAliases(t *testing.T) {
	m := git.NewMemoryRunner()
	m.GlobalConfig[config.ElegantAliasKey] = "elegant"
	m.GlobalConfig["alias.start-work"] = "elegant start-work"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	findings := GitInstall()
	var alias Finding
	for _, f := range findings {
		if strings.Contains(f.Problem, "aliases") {
			alias = f
		}
	}
	if alias.Apply == nil {
		t.Fatalf("expected alias finding, got %v", problems(findings))
	}
	if err := alias.Apply(); err != nil {
		t.Fatal(err)
	}
	if m.GlobalConfig[config.ElegantAliasKey] != config.ElegantAliasValue {
		t.Fatalf("alias.elegant = %q", m.GlobalConfig[config.ElegantAliasKey])
	}
	if m.GlobalConfig["alias.start-work"] != "!eg work start" {
		t.Fatalf("alias.start-work = %q", m.GlobalConfig["alias.start-work"])
	}
}

func TestGitInstallAdvisoryCompletionAndBinary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	comp := filepath.Join(home, ".local/share/bash-completion/completions/git-elegant")
	if err := os.MkdirAll(filepath.Dir(comp), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(comp, []byte("# old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	legacyBin := filepath.Join(binDir, "git-elegant")
	if err := os.WriteFile(legacyBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	m := git.NewMemoryRunner()
	m.GlobalConfig[config.ElegantAliasKey] = config.ElegantAliasValue
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	findings := GitInstall()
	var completion, binary bool
	for _, f := range findings {
		if strings.Contains(f.Problem, "completion") {
			completion = true
			if f.Apply != nil {
				t.Fatal("completion must be advisory")
			}
		}
		if strings.Contains(f.Problem, "git-elegant is still on PATH") {
			binary = true
			if f.Apply != nil {
				t.Fatal("binary must be advisory")
			}
		}
	}
	if !completion || !binary {
		t.Fatalf("completion=%v binary=%v findings=%v", completion, binary, problems(findings))
	}
}

func TestGitInstallFreshMachineHasNoAliasFinding(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	m := git.NewMemoryRunner()
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	for _, f := range GitInstall() {
		if strings.Contains(f.Problem, "aliases") {
			t.Fatalf("fresh machine must not flag aliases: %v", problems(GitInstall()))
		}
	}
}

func problems(findings []Finding) []string {
	out := make([]string, len(findings))
	for i, f := range findings {
		out[i] = f.Problem
	}
	return out
}
