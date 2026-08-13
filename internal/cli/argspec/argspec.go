// Package argspec implements the uniform required/optional argument policy.
package argspec

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

// Kind distinguishes positional arguments from flags.
type Kind int

const (
	Positional Kind = iota
	Flag
)

const maxPickRetries = 3

// Choice is one completion value with an optional description.
type Choice struct {
	Value       string
	Description string
}

// CompleteFunc returns selectable values for an input.
type CompleteFunc func(ctx context.Context) ([]Choice, error)

// Input describes one command input resolved before business logic runs.
type Input struct {
	Name        string
	Kind        Kind
	Label       string
	Required    bool
	Index       int
	FlagName    string
	Get         func() string
	Set         func(string)
	Suggest     func() string
	Validate    func(string) error
	Complete    CompleteFunc
	ValidateCLI bool // when false, CLI args skip completion membership (interactive-only)
	// OmitInteractive skips prompting for an optional input when unset (empty means "all").
	OmitInteractive bool
}

// Spec is the full argument specification for a command.
type Spec struct {
	Inputs []Input
}

// ErrMissingRequired is returned in non-interactive mode when required inputs are unset.
type ErrMissingRequired struct {
	Names []string
}

func (e *ErrMissingRequired) Error() string {
	return fmt.Sprintf("required input missing in non-interactive mode: %s", strings.Join(e.Names, ", "))
}

// ErrEmptySource is returned when Complete yields no values.
type ErrEmptySource struct {
	Name string
}

func (e *ErrEmptySource) Error() string {
	return fmt.Sprintf("no values available for %s", e.Name)
}

// IsEmptySource reports whether err is ErrEmptySource.
func IsEmptySource(err error) bool {
	var es *ErrEmptySource
	return errors.As(err, &es)
}

// ResolveCmd resolves inputs using the prompter from cmd.Context().
func ResolveCmd(cmd *cobra.Command, args []string, spec Spec) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	return Resolve(ctx, prompt.FromContext(ctx), args, spec)
}

// Resolve applies the argument policy after cobra has parsed flags and args.
func Resolve(ctx context.Context, p prompt.Prompter, args []string, spec Spec) error {
	for i := range spec.Inputs {
		in := &spec.Inputs[i]
		if in.Kind == Positional && in.Index < len(args) && strings.TrimSpace(in.Get()) == "" {
			in.Set(args[in.Index])
		}
	}

	for _, in := range spec.Inputs {
		val := strings.TrimSpace(in.Get())
		if val == "" || in.Complete == nil || !in.ValidateCLI {
			continue
		}
		if err := validateChoice(ctx, in, val); err != nil {
			return err
		}
	}

	var missing []Input
	for _, in := range spec.Inputs {
		if in.Required && strings.TrimSpace(in.Get()) == "" {
			missing = append(missing, in)
		}
	}
	if len(missing) > 0 && prompt.NonInteractive(p) {
		names := make([]string, len(missing))
		for i, in := range missing {
			names[i] = in.Name
		}
		return &ErrMissingRequired{Names: names}
	}

	for _, in := range missing {
		val, err := resolveValue(ctx, p, in, "")
		if err != nil {
			return err
		}
		in.Set(val)
	}

	if prompt.NonInteractive(p) {
		return nil
	}

	for _, in := range spec.Inputs {
		if in.Required {
			continue
		}
		current := strings.TrimSpace(in.Get())
		if in.OmitInteractive {
			continue
		}
		if in.Kind == Positional {
			if in.Index < len(args) {
				continue
			}
			if allRequiredPositionalsFromCLI(args, spec) {
				continue
			}
		}
		suggested := current
		if suggested == "" && in.Suggest != nil {
			suggested = in.Suggest()
		}
		val, err := resolveValue(ctx, p, in, suggested)
		if err != nil {
			if errors.Is(err, prompt.ErrUserCancelled) && current != "" {
				continue
			}
			return err
		}
		if in.Validate != nil {
			if vErr := in.Validate(val); vErr != nil {
				return vErr
			}
		}
		in.Set(val)
	}
	return nil
}

func allRequiredPositionalsFromCLI(args []string, spec Spec) bool {
	for _, in := range spec.Inputs {
		if !in.Required || in.Kind != Positional {
			continue
		}
		if in.Index >= len(args) || strings.TrimSpace(args[in.Index]) == "" {
			return false
		}
	}
	return true
}

func resolveValue(ctx context.Context, p prompt.Prompter, in Input, suggested string) (string, error) {
	if in.Complete != nil {
		return pickFromComplete(ctx, p, in, suggested)
	}
	if suggested != "" {
		return p.EditOrAccept(in.Label, suggested)
	}
	val, err := p.String(promptFor(in), "")
	if err != nil {
		return "", err
	}
	val = strings.TrimSpace(val)
	if in.Validate != nil {
		if vErr := in.Validate(val); vErr != nil {
			return "", vErr
		}
	}
	if val == "" && in.Required {
		return "", fmt.Errorf("%s is required", in.Name)
	}
	return val, nil
}

func pickFromComplete(ctx context.Context, p prompt.Prompter, in Input, suggested string) (string, error) {
	choices, err := in.Complete(ctx)
	if err != nil {
		return "", err
	}
	if len(choices) == 0 {
		return "", &ErrEmptySource{Name: in.Name}
	}
	promptChoices := toPromptChoices(choices)
	var lastVal string
	for attempt := 0; attempt < maxPickRetries; attempt++ {
		val, err := p.Pick(in.Label, promptChoices)
		if err != nil {
			if errors.Is(err, prompt.ErrUserCancelled) {
				if suggested != "" && memberOf(choices, suggested) {
					return suggested, nil
				}
				return "", err
			}
			return "", err
		}
		val = strings.TrimSpace(val)
		if i := strings.IndexByte(val, '\t'); i >= 0 {
			val = val[:i]
		}
		lastVal = val
		if memberOf(choices, val) {
			if in.Validate != nil {
				if vErr := in.Validate(val); vErr != nil {
					return "", vErr
				}
			}
			return val, nil
		}
	}
	return "", fmt.Errorf("%s: %q is not a valid choice", in.Name, lastVal)
}

func validateChoice(ctx context.Context, in Input, val string) error {
	choices, err := in.Complete(ctx)
	if err != nil {
		return err
	}
	if len(choices) == 0 {
		return &ErrEmptySource{Name: in.Name}
	}
	if !memberOf(choices, val) {
		return fmt.Errorf("%s: %q is not a valid choice", in.Name, val)
	}
	if in.Validate != nil {
		return in.Validate(val)
	}
	return nil
}

func memberOf(choices []Choice, val string) bool {
	for _, c := range choices {
		if c.Value == val {
			return true
		}
	}
	return false
}

func toPromptChoices(choices []Choice) []prompt.Choice {
	out := make([]prompt.Choice, len(choices))
	for i, c := range choices {
		out[i] = prompt.Choice{Value: c.Value, Description: c.Description}
	}
	return out
}

func promptFor(in Input) string {
	if in.Label != "" {
		return in.Label
	}
	return "What is your " + in.Name + "?"
}

// PositionalInput builds a positional input bound to ptr.
func PositionalInput(name string, index int, required bool, label string, ptr *string, suggest func() string) Input {
	return Input{
		Name:        name,
		Kind:        Positional,
		Label:       label,
		Required:    required,
		Index:       index,
		Get:         func() string { return *ptr },
		Set:         func(v string) { *ptr = v },
		Suggest:     suggest,
		ValidateCLI: true,
	}
}

// PositionalInputWithComplete adds a completion source to a positional input.
func PositionalInputWithComplete(name string, index int, required bool, label string, ptr *string, suggest func() string, complete CompleteFunc, validateCLI bool) Input {
	in := PositionalInput(name, index, required, label, ptr, suggest)
	in.Complete = complete
	in.ValidateCLI = validateCLI
	return in
}

// IsMissingRequired reports whether err is ErrMissingRequired.
func IsMissingRequired(err error) bool {
	var mr *ErrMissingRequired
	return errors.As(err, &mr)
}
