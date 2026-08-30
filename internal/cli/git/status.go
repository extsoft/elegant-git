package git

import (
	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	c := &cobra.Command{
		Use:    "status",
		Hidden: true,
		Short:  "Show global Git installation and shared memory state",
		Long:   "Prints global Git configuration, elegant-git memory paths, and workspace/repository counts.",
		RunE:   runList,
	}
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[deprecation.SurfaceAnnotation] = "git status"
	return c
}
