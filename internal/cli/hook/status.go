package hook

import (
	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	c := &cobra.Command{
		Use:    "status",
		Hidden: true,
		Short:  "Lists configured hook file paths",
		Long:   "Prints paths of ahead/after hook scripts (new and legacy layouts).",
		RunE:   runList,
	}
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[deprecation.SurfaceAnnotation] = "hook status"
	return c
}
