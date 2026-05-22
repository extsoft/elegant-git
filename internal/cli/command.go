package cli

import "github.com/spf13/cobra"

var implementedCommands = map[string]func(commandSpec) *cobra.Command{
	"acquire-git":        newAcquireGitCommand,
	"acquire-repository": newAcquireRepositoryCommand,
	"clone-repository":   newCloneRepositoryCommand,
	"init-repository":    newInitRepositoryCommand,
	"prune-repository":   newPruneRepositoryCommand,
}

func newCommand(spec commandSpec) *cobra.Command {
	if ctor, ok := implementedCommands[spec.name]; ok {
		return ctor(spec)
	}
	return newStubCommand(spec)
}
