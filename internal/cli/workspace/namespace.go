package workspace

import (
	"fmt"

	"github.com/bees-hive/elegant-git/internal/giturl"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/prompt"
)

// CaptureNamespace records origin's namespace on the workspace after
// confirmation. Non-interactive mode records silently. No-op when origin has
// no namespace or the workspace already has it.
func CaptureNamespace(s *shared.State, workspaceID string, ws *shared.Workspace, origin string, p prompt.Prompter) error {
	ns, ok := giturl.Namespace(origin)
	if !ok {
		return nil
	}
	for _, n := range ws.Namespaces {
		if n == ns {
			return nil
		}
	}
	if !prompt.NonInteractive(p) {
		ok, err := p.Confirm(fmt.Sprintf("Remember %q as a namespace of workspace %q?", ns, ws.Name), true)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
	_, err := shared.AddWorkspaceNamespace(s, workspaceID, ns)
	return err
}
