package runtime

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// UsageError carries a command whose help should be shown after a usage mistake.
type UsageError struct {
	Cmd *cobra.Command
	Err error
}

func (e *UsageError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "invalid usage"
}

// NewUsageError wraps err as a usage error for cmd.
func NewUsageError(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}
	return &UsageError{Cmd: cmd, Err: err}
}

// IsUsageError reports whether err is a UsageError.
func IsUsageError(err error) bool {
	var ue *UsageError
	return errors.As(err, &ue)
}

// FlagErrorWithHelp wraps unknown or invalid flags as UsageError.
func FlagErrorWithHelp(cmd *cobra.Command, err error) error {
	return NewUsageError(cmd, err)
}

// EmitCommandHelp invokes cmd's HelpFunc writing to w.
func EmitCommandHelp(w io.Writer, cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	oldOut := cmd.OutOrStdout()
	cmd.SetOut(w)
	defer cmd.SetOut(oldOut)
	if fn := cmd.HelpFunc(); fn != nil {
		fn(cmd, nil)
	}
}

// RejectExtraArgs rejects unexpected positional arguments.
func RejectExtraArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return NewUsageError(cmd, fmt.Errorf("accepts no arguments, received %d", len(args)))
	}
	return nil
}

// ConfigureCommandTree sets help and usage-error handling on root and descendants.
func ConfigureCommandTree(root *cobra.Command) {
	configureCommandRecursive(root)
}

func configureCommandRecursive(c *cobra.Command) {
	c.SetFlagErrorFunc(FlagErrorWithHelp)
	if len(c.Commands()) == 0 {
		c.SetHelpFunc(CommandHelp)
		if c.Args == nil {
			c.Args = RejectExtraArgs
		}
		return
	}
	for _, child := range c.Commands() {
		configureCommandRecursive(child)
	}
}
