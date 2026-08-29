package workspace

import (
	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/spf13/cobra"
)

// NewLegacyProfileCommand returns a hidden `profile` alias of the workspace object.
func NewLegacyProfileCommand() *cobra.Command {
	c := NewCommand()
	c.Use = "profile"
	c.Hidden = true
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[deprecation.SurfaceAnnotation] = "profile"
	for _, sub := range c.Commands() {
		annotateLegacy(sub)
	}
	return c
}

func annotateLegacy(c *cobra.Command) {
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[deprecation.SurfaceAnnotation] = "profile"
	for _, sub := range c.Commands() {
		annotateLegacy(sub)
	}
}
