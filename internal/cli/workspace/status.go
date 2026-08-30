package workspace

import (
	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	c := &cobra.Command{
		Use:    "status",
		Hidden: true,
		Short:  "Show the linked workspace for the current repository",
		Long:   "Prints the workspace linked to the current repository from shared memory.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return PrintWorkspaceStatus(cmd.OutOrStdout())
		},
	}
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[deprecation.SurfaceAnnotation] = "workspace status"
	return c
}
