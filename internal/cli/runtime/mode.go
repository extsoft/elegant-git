package runtime

import (
	"io"
	"os"
	"strings"

	"github.com/extsoft/elegant-git/internal/prompt"
)

// InteractionMode is whether the CLI may prompt for input.
type InteractionMode int

const (
	ModeInteractive InteractionMode = iota
	ModeNonInteractive
)

// ModeConfig carries explicit overrides for interaction mode resolution.
type ModeConfig struct {
	ForceInteractive    bool
	ForceNonInteractive bool
	Stdin               io.Reader
	// TTY overrides stdin terminal detection when non-nil.
	TTY *bool
}

// ResolveMode picks interactive vs non-interactive. Non-TTY stdin always
// wins; flags and CI apply only when stdin is a TTY.
func ResolveMode(cfg ModeConfig) InteractionMode {
	if !cfg.isTTY() {
		return ModeNonInteractive
	}
	if cfg.ForceInteractive || envTruthy("ELEGANT_GIT_INTERACTIVE") {
		return ModeInteractive
	}
	if cfg.ForceNonInteractive || envTruthy("ELEGANT_GIT_NON_INTERACTIVE") {
		return ModeNonInteractive
	}
	if envTruthy("CI") {
		return ModeNonInteractive
	}
	return ModeInteractive
}

func (cfg ModeConfig) isTTY() bool {
	if cfg.TTY != nil {
		return *cfg.TTY
	}
	return stdinIsTTY(cfg.Stdin)
}

// PrompterForMode returns the prompter for the resolved mode.
func PrompterForMode(mode InteractionMode, in io.Reader, out io.Writer) prompt.Prompter {
	if mode == ModeNonInteractive {
		return prompt.NewNonInteractive()
	}
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	return prompt.NewTTY(in, out)
}

func envTruthy(key string) bool {
	v := strings.TrimSpace(os.Getenv(key))
	return v == "1" || strings.EqualFold(v, "true")
}

func stdinIsTTY(r io.Reader) bool {
	if r == nil {
		r = os.Stdin
	}
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
