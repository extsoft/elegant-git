package runtime

import (
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

// PickActionOrHelp prompts until the user picks an action, quits, or views help.
func PickActionOrHelp(cmd *cobra.Command, p prompt.Prompter, label string, choices []prompt.Choice, def string) (string, error) {
	for {
		ans, err := p.Pick(label, choices, def)
		if err != nil {
			return "", err
		}
		if ans == "" || ans == "quit" {
			return ans, nil
		}
		if ans == "help" {
			if err := cmd.Help(); err != nil {
				return "", err
			}
			continue
		}
		return ans, nil
	}
}
