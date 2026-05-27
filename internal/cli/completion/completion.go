package completion

import (
	"fmt"
	"os"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/spf13/cobra"
)

// NewCommand returns the completion subcommand.
func NewCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long:  "The completion command generates shell completion scripts for bash, zsh, fish, or powershell.",
		RunE: func(cmd *cobra.Command, args []string) error {
			var shell string
			if err := argspec.ResolveCmd(cmd, args, argspec.Spec{Inputs: []argspec.Input{
				argspec.PositionalInput("shell", 0, true, "Shell (bash, zsh, fish, powershell)", &shell, nil),
			}}); err != nil {
				return err
			}
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
	return c
}
