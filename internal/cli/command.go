package cli

import "github.com/spf13/cobra"

var implementedCommands = map[string]func(commandSpec) *cobra.Command{
	"acquire-git":        newAcquireGitCommand,
	"acquire-repository": newAcquireRepositoryCommand,
	"clone-repository":   newCloneRepositoryCommand,
	"init-repository":    newInitRepositoryCommand,
	"prune-repository":   newPruneRepositoryCommand,
	"show-commands":      newShowCommandsCommand,
	"show-workflows":     newShowWorkflowsCommand,
	"make-workflow":      newMakeWorkflowCommand,
	"polish-workflow":    newPolishWorkflowCommand,
	"start-work":         newStartWorkCommand,
	"save-work":          newSaveWorkCommand,
	"amend-work":         newAmendWorkCommand,
	"show-work":          newShowWorkCommand,
	"polish-work":        newPolishWorkCommand,
	"actualize-work":     newActualizeWorkCommand,
	"deliver-work":       newDeliverWorkCommand,
	"obtain-work":        newObtainWorkCommand,
	"accept-work":        newAcceptWorkCommand,
	"release-work":       newReleaseWorkCommand,
	"show-release-notes": newShowReleaseNotesCommand,
}

func newCommand(spec commandSpec) *cobra.Command {
	if ctor, ok := implementedCommands[spec.name]; ok {
		return ctor(spec)
	}
	return newStubCommand(spec)
}
