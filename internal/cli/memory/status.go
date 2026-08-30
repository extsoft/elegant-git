package memory

import (
	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	c := &cobra.Command{
		Use:    "status",
		Hidden: true,
		Short:  "Summarize elegant-git memory stores",
		Long:   "Prints shared memory paths, workspace and repository counts, and hints for detail commands.",
		RunE:   runList,
	}
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[deprecation.SurfaceAnnotation] = "memory status"
	return c
}
