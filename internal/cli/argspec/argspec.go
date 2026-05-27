// Package argspec implements the uniform required/optional argument policy.
package argspec

import (
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

// Input describes one command input resolved before business logic runs.
type Input struct {
	Name     string
	Kind     Kind
	Label    string
	Required bool
	Index    int
	FlagName string
	Get      func() string
	Set      func(string)
	Suggest  func() string
	Validate func(string) error
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

// ResolveCmd resolves inputs using the prompter from cmd.Context().
func ResolveCmd(cmd *cobra.Command, args []string, spec Spec) error {
	return Resolve(prompt.FromContext(cmd.Context()), args, spec)
}

// Resolve applies the argument policy after cobra has parsed flags and args.
func Resolve(p prompt.Prompter, args []string, spec Spec) error {
	for i := range spec.Inputs {
		in := &spec.Inputs[i]
		if in.Kind == Positional && in.Index < len(args) && strings.TrimSpace(in.Get()) == "" {
			in.Set(args[in.Index])
		}
	}

	var missing []Input
	for _, in := range spec.Inputs {
		if in.Required && strings.TrimSpace(in.Get()) == "" {
			missing = append(missing, in)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	if prompt.NonInteractive(p) {
		names := make([]string, len(missing))
		for i, in := range missing {
			names[i] = in.Name
		}
		return &ErrMissingRequired{Names: names}
	}

	for _, in := range missing {
		val, err := p.String(promptFor(in), "")
		if err != nil {
			return err
		}
		val = strings.TrimSpace(val)
		if in.Validate != nil {
			if vErr := in.Validate(val); vErr != nil {
				return vErr
			}
		}
		if val == "" {
			return fmt.Errorf("%s is required", in.Name)
		}
		in.Set(val)
	}

	for _, in := range spec.Inputs {
		if in.Required {
			continue
		}
		current := strings.TrimSpace(in.Get())
		suggested := current
		if suggested == "" && in.Suggest != nil {
			suggested = in.Suggest()
		}
		val, err := p.EditOrAccept(in.Label, suggested)
		if err != nil {
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

func promptFor(in Input) string {
	if in.Label != "" {
		return in.Label
	}
	return "What is your " + in.Name + "?"
}

// PositionalInput builds a positional input bound to ptr.
func PositionalInput(name string, index int, required bool, label string, ptr *string, suggest func() string) Input {
	return Input{
		Name:     name,
		Kind:     Positional,
		Label:    label,
		Required: required,
		Index:    index,
		Get:      func() string { return *ptr },
		Set:      func(v string) { *ptr = v },
		Suggest:  suggest,
	}
}

// FlagInput builds a flag input bound to ptr.
func FlagInput(name, flagName string, required bool, label string, ptr *string, suggest func() string) Input {
	return Input{
		Name:     name,
		Kind:     Flag,
		Label:    label,
		Required: required,
		FlagName: flagName,
		Get:      func() string { return *ptr },
		Set:      func(v string) { *ptr = v },
		Suggest:  suggest,
	}
}

// IsMissingRequired reports whether err is ErrMissingRequired.
func IsMissingRequired(err error) bool {
	var mr *ErrMissingRequired
	return errors.As(err, &mr)
}
