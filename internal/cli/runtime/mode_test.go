package runtime

import (
	"strings"
	"testing"

	"github.com/bees-hive/elegant-git/internal/prompt"
)

func TestResolveModePrecedence(t *testing.T) {
	yes, no := true, false
	tests := []struct {
		name string
		cfg  ModeConfig
		env  map[string]string
		want InteractionMode
	}{
		{
			name: "non-tty wins over ForceInteractive",
			cfg:  ModeConfig{ForceInteractive: true, TTY: &no},
			want: ModeNonInteractive,
		},
		{
			name: "non-tty wins over ELEGANT_GIT_INTERACTIVE",
			cfg:  ModeConfig{TTY: &no},
			env:  map[string]string{"ELEGANT_GIT_INTERACTIVE": "1"},
			want: ModeNonInteractive,
		},
		{
			name: "non-tty stdin reader is non-interactive",
			cfg:  ModeConfig{Stdin: strings.NewReader("")},
			env:  map[string]string{"ELEGANT_GIT_INTERACTIVE": "1"},
			want: ModeNonInteractive,
		},
		{
			name: "tty ForceInteractive beats CI and non-interactive",
			cfg:  ModeConfig{ForceInteractive: true, TTY: &yes},
			env: map[string]string{
				"CI":                          "true",
				"ELEGANT_GIT_NON_INTERACTIVE": "1",
			},
			want: ModeInteractive,
		},
		{
			name: "tty ELEGANT_GIT_INTERACTIVE beats non-interactive env",
			cfg:  ModeConfig{TTY: &yes},
			env: map[string]string{
				"ELEGANT_GIT_INTERACTIVE":     "1",
				"ELEGANT_GIT_NON_INTERACTIVE": "1",
			},
			want: ModeInteractive,
		},
		{
			name: "tty ForceNonInteractive",
			cfg:  ModeConfig{ForceNonInteractive: true, TTY: &yes},
			want: ModeNonInteractive,
		},
		{
			name: "tty ELEGANT_GIT_NON_INTERACTIVE",
			cfg:  ModeConfig{TTY: &yes},
			env:  map[string]string{"ELEGANT_GIT_NON_INTERACTIVE": "1"},
			want: ModeNonInteractive,
		},
		{
			name: "tty CI",
			cfg:  ModeConfig{TTY: &yes},
			env:  map[string]string{"CI": "true"},
			want: ModeNonInteractive,
		},
		{
			name: "tty default interactive",
			cfg:  ModeConfig{TTY: &yes},
			want: ModeInteractive,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CI", "")
			t.Setenv("ELEGANT_GIT_INTERACTIVE", "")
			t.Setenv("ELEGANT_GIT_NON_INTERACTIVE", "")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			if got := ResolveMode(tc.cfg); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestPrompterForMode(t *testing.T) {
	p := PrompterForMode(ModeNonInteractive, nil, nil)
	if !prompt.NonInteractive(p) {
		t.Fatal("expected non-interactive prompter")
	}
}
