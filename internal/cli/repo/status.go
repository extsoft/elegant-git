package repo

import (
	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	c := &cobra.Command{
		Use:    "status",
		Hidden: true,
		Short:  "Show repository memory and registry state for the current repository",
		Long:   "Prints per-repository memory, registry linkage, and workspace association for the current repository.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return printRepoState(cmd.OutOrStdout())
		},
	}
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[deprecation.SurfaceAnnotation] = "repo status"
	return c
}
