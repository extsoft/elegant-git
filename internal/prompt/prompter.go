package prompt

import (
	"context"
	"errors"
	"fmt"
)

// ErrNonInteractive is returned when input is required but prompting is disabled.
var ErrNonInteractive = errors.New("required input missing in non-interactive mode")

type ctxKey struct{}

// BatchDecision is the result of a multi-repo propagation prompt.
type BatchDecision int

const (
	BatchConfirm BatchDecision = iota
	BatchReject
	BatchApplyAll
	BatchSkip
)

// ErrUserCancelled is returned when the user aborts a picker (e.g. Esc).
var ErrUserCancelled = errors.New("input cancelled")

// Prompter abstracts interactive CLI input.
type Prompter interface {
	String(question, defaultVal string) (string, error)
	Confirm(question string, defaultYes bool) (bool, error)
	Choose(question string, options []string) (int, error)
	Pick(label string, choices []Choice) (string, error)
	Required(label, current string) error
	EditOrAccept(label, suggested string) (string, error)
	Optional(label, suggested string) (string, error)
	Closed(question string, options []string, defaultWord string, required bool) (string, error)
	BatchChoice(question, defaultWord string) (BatchDecision, error)
}

// Skippable returns current when it equals suggested; otherwise EditOrAccept.
func Skippable(p Prompter, label, current, suggested string) (string, error) {
	if current == suggested {
		return current, nil
	}
	return p.EditOrAccept(label, suggested)
}

// NonInteractive reports whether p rejects interactive prompts.
func NonInteractive(p Prompter) bool {
	_, ok := p.(*nonInteractive)
	return ok
}

// WithPrompter stores a Prompter on ctx.
func WithPrompter(ctx context.Context, p Prompter) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// FromContext returns the Prompter from ctx or a TTY prompter using stdin/stdout.
func FromContext(ctx context.Context) Prompter {
	if p, ok := ctx.Value(ctxKey{}).(Prompter); ok && p != nil {
		return p
	}
	return NewTTY(nil, nil)
}

// RequireString returns flag value or prompts; errors in non-interactive mode when empty.
func RequireString(p Prompter, flagVal, label string) (string, error) {
	if flagVal != "" {
		return flagVal, nil
	}
	if NonInteractive(p) {
		return "", fmt.Errorf("%s is required in non-interactive mode", label)
	}
	return p.String("What is your "+label+"?", "")
}
