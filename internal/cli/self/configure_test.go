package self

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func TestConfigureRunOrder(t *testing.T) {
	m, cmd := setupConfigure(t)
	m.GlobalConfig["user.name"] = "Alice"
	m.GlobalConfig["user.email"] = "alice@example.com"
	m.GlobalConfig["alias.start-work"] = "!eg work start"

	var buf bytes.Buffer
	text.SetOutput(&buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewTTY(strings.NewReader("\n\n\n"), &buf)))

	if err := configureRun(cmd); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	assertContainsOrder(t, out,
		"Starting global Git configuration",
		"Git basics",
		"Git standards",
		"Git aliases",
		"Press enter to continue...",
		"Configuring basics",
		"Configuring standards",
		"Removing obsolete configurations",
		"Configuring aliases",
		"Global Git configuration complete",
	)
	if strings.Contains(out, "Press enter to continue...:") {
		t.Fatal("continue must not use a closed-question colon")
	}
	if strings.Contains(out, "Would you like to apply a global configuration") {
		t.Fatal("old opt-out question must be gone")
	}

	unsetAt, setAt := -1, -1
	for i, c := range m.Calls {
		if len(c.Args) < 4 || c.Args[0] != "config" || c.Args[1] != "--global" {
			continue
		}
		if c.Args[2] == "--unset" && c.Args[3] == "alias.start-work" {
			unsetAt = i
		}
		if c.Args[2] == "alias.start-work" {
			setAt = i
		}
	}
	if unsetAt < 0 || setAt < 0 {
		t.Fatalf("expected alias remove then add; unset=%d set=%d calls=%v", unsetAt, setAt, m.Calls)
	}
	if unsetAt > setAt {
		t.Fatalf("alias remove at %d after add at %d", unsetAt, setAt)
	}
	if !config.IsGitAcquired() {
		t.Fatal("expected acquired after configure")
	}
	if m.GlobalConfig["pull.rebase"] != "true" {
		t.Fatalf("standards not applied: pull.rebase=%q", m.GlobalConfig["pull.rebase"])
	}
}

func TestConfigureRequiresUserIdentity(t *testing.T) {
	m, cmd := setupConfigure(t)
	var buf bytes.Buffer
	text.SetOutput(&buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	in := strings.NewReader("\n\nAlice\nalice@example.com\n\n\n")
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewTTY(in, &buf)))
	if err := configureRun(cmd); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "Git user.name (press enter to skip)") {
		t.Fatal("user.name must be required")
	}
	if strings.Contains(out, "Git user.email (press enter to skip)") {
		t.Fatal("user.email must be required")
	}
	if !strings.Contains(out, "Git user.name:") {
		t.Fatalf("expected required name prompt:\n%s", out)
	}
	if m.GlobalConfig["user.name"] != "Alice" || m.GlobalConfig["user.email"] != "alice@example.com" {
		t.Fatalf("identity = %q %q", m.GlobalConfig["user.name"], m.GlobalConfig["user.email"])
	}
}

func TestAutoConfigureSkipsWhenAcquired(t *testing.T) {
	m, configure := setupConfigure(t)
	listCmd := newListCommand()
	configure.Parent().AddCommand(listCmd)
	s, err := shared.Load()
	if err != nil {
		t.Fatal(err)
	}
	shared.SetAcquired(s, "test")
	if err := shared.Save(s); err != nil {
		t.Fatal(err)
	}
	listCmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewTTY(strings.NewReader("\n"), &bytes.Buffer{})))
	if err := AutoConfigure(listCmd); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.GlobalConfig["pull.rebase"]; ok {
		t.Fatal("must not apply standards when already acquired")
	}
}

func TestAutoConfigureSkipsSelfConfigure(t *testing.T) {
	m, cmd := setupConfigure(t)
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewTTY(strings.NewReader("\n"), &bytes.Buffer{})))
	if err := AutoConfigure(cmd); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.GlobalConfig["pull.rebase"]; ok {
		t.Fatal("must not run configure from AutoConfigure on self configure")
	}
	if config.IsGitAcquired() {
		t.Fatal("must not mark acquired")
	}
}

func TestAutoConfigureSkipsGitConfigureShim(t *testing.T) {
	m, _ := setupConfigure(t)
	gitCmd := &cobra.Command{Use: "git"}
	cmd := &cobra.Command{Use: "configure"}
	gitCmd.AddCommand(cmd)
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewTTY(strings.NewReader("\n"), &bytes.Buffer{})))
	if err := AutoConfigure(cmd); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.GlobalConfig["pull.rebase"]; ok {
		t.Fatal("must not run configure from AutoConfigure on git configure")
	}
}

func TestAutoConfigureSkipsAcquireGit(t *testing.T) {
	m, _ := setupConfigure(t)
	cmd := &cobra.Command{Use: "acquire-git"}
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewTTY(strings.NewReader("\n"), &bytes.Buffer{})))
	if err := AutoConfigure(cmd); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.GlobalConfig["pull.rebase"]; ok {
		t.Fatal("must not run configure from AutoConfigure on acquire-git")
	}
}

func TestAutoConfigureSkipsNonInteractive(t *testing.T) {
	m, configure := setupConfigure(t)
	listCmd := newListCommand()
	configure.Parent().AddCommand(listCmd)
	listCmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := AutoConfigure(listCmd); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.GlobalConfig["pull.rebase"]; ok {
		t.Fatal("must not apply standards in non-interactive auto-configure")
	}
}

func TestAutoConfigureThenList(t *testing.T) {
	m, configure := setupConfigure(t)
	m.GlobalConfig["user.name"] = "Alice"
	m.GlobalConfig["user.email"] = "alice@example.com"
	listCmd := newListCommand()
	configure.Parent().AddCommand(listCmd)

	var buf bytes.Buffer
	text.SetOutput(&buf)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	listCmd.SetOut(&buf)
	listCmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewTTY(strings.NewReader("\n\n\n"), &buf)))

	if err := AutoConfigure(listCmd); err != nil {
		t.Fatal(err)
	}
	if !config.IsGitAcquired() {
		t.Fatal("expected configure to run")
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Global git identity") {
		t.Fatalf("list missing identity section:\n%s", buf.String())
	}
}

func TestConfigureContinueAcceptsAnyInput(t *testing.T) {
	m, cmd := setupConfigure(t)
	m.GlobalConfig["user.name"] = "Alice"
	m.GlobalConfig["user.email"] = "alice@example.com"
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewTTY(strings.NewReader("yes\n\n\n"), &bytes.Buffer{})))
	if err := configureRun(cmd); err != nil {
		t.Fatal(err)
	}
	if !config.IsGitAcquired() {
		t.Fatal("expected acquired")
	}
}

func TestConfigureNonInteractiveRequiresIdentity(t *testing.T) {
	m, cmd := setupConfigure(t)
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := configureRun(cmd); err == nil {
		t.Fatal("expected error when name and email are missing")
	}
	if config.IsGitAcquired() {
		t.Fatal("must not mark acquired without identity")
	}
	if _, ok := m.GlobalConfig["pull.rebase"]; ok {
		t.Fatal("must not apply standards without identity")
	}
}

func TestEnsureConfiguredNonInteractiveApplies(t *testing.T) {
	m, cmd := setupConfigure(t)
	m.GlobalConfig["user.name"] = "Alice"
	m.GlobalConfig["user.email"] = "alice@example.com"
	cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := configureRun(cmd); err != nil {
		t.Fatal(err)
	}
	if !config.IsGitAcquired() {
		t.Fatal("explicit configure still applies when non-interactive")
	}
	if m.GlobalConfig["pull.rebase"] != "true" {
		t.Fatal("standards not applied")
	}
}

func setupConfigure(t *testing.T) (*git.MemoryRunner, *cobra.Command) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("ELEGANT_GIT_STATE_FILE", filepath.Join(dir, "state.json"))
	m := git.NewMemoryRunner()
	git.Use(m)
	selfCmd := &cobra.Command{Use: "self"}
	c := newConfigureCommand()
	selfCmd.AddCommand(c)
	return m, c
}

func assertContainsOrder(t *testing.T, s string, parts ...string) {
	t.Helper()
	last := -1
	for _, p := range parts {
		i := strings.Index(s, p)
		if i < 0 {
			t.Fatalf("missing %q in:\n%s", p, s)
		}
		if i < last {
			t.Fatalf("%q at %d before previous at %d", p, i, last)
		}
		last = i
	}
}
