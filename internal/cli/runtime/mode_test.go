package runtime

import (
	"strings"
	"testing"

	"github.com/bees-hive/elegant-git/internal/prompt"
)

func TestResolveModePrecedence(t *testing.T) {
	t.Setenv("CI", "true")
	t.Setenv("ELEGANT_GIT_NON_INTERACTIVE", "1")
	cfg := ModeConfig{ForceInteractive: true}
	if got := ResolveMode(cfg); got != ModeInteractive {
		t.Fatalf("ForceInteractive: got %v", got)
	}

	t.Setenv("ELEGANT_GIT_INTERACTIVE", "1")
	t.Setenv("ELEGANT_GIT_NON_INTERACTIVE", "1")
	if got := ResolveMode(ModeConfig{}); got != ModeInteractive {
		t.Fatalf("ELEGANT_GIT_INTERACTIVE: got %v", got)
	}

	t.Setenv("ELEGANT_GIT_INTERACTIVE", "")
	cfg = ModeConfig{ForceNonInteractive: true}
	if got := ResolveMode(cfg); got != ModeNonInteractive {
		t.Fatalf("ForceNonInteractive: got %v", got)
	}

	t.Setenv("ELEGANT_GIT_INTERACTIVE", "")
	t.Setenv("ELEGANT_GIT_NON_INTERACTIVE", "1")
	if got := ResolveMode(ModeConfig{}); got != ModeNonInteractive {
		t.Fatalf("ELEGANT_GIT_NON_INTERACTIVE: got %v", got)
	}

	t.Setenv("ELEGANT_GIT_NON_INTERACTIVE", "")
	t.Setenv("CI", "true")
	if got := ResolveMode(ModeConfig{Stdin: strings.NewReader("")}); got != ModeNonInteractive {
		t.Fatalf("CI: got %v", got)
	}

	t.Setenv("CI", "")
	if got := ResolveMode(ModeConfig{Stdin: strings.NewReader("")}); got != ModeNonInteractive {
		t.Fatalf("non-tty stdin: got %v", got)
	}
}

func TestPrompterForMode(t *testing.T) {
	p := PrompterForMode(ModeNonInteractive, nil, nil)
	if !prompt.NonInteractive(p) {
		t.Fatal("expected non-interactive prompter")
	}
}
