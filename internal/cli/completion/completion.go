package completion

import (
	"fmt"
	"os"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/cli/sources"
	"github.com/spf13/cobra"
)

// NewCommand returns the completion subcommand.
func NewCommand() *cobra.Command {
	var shell string
	spec := argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInputWithComplete("shell", 0, true, "Completion shell", &shell, nil, sources.CompletionShells, true).AsClosed(),
	}}
	c := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long:  "The completion command generates shell completion scripts for bash, zsh, fish, or powershell.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
				return err
			}
			shell = spec.Inputs[0].Get()
			switch shell {
			case "bash":
				return cmd.Root().GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell %q (bash, zsh, fish, powershell)", shell)
			}
		},
	}
	Attach(c, spec)
	return c
}
