package git

import (
	"fmt"

	"github.com/extsoft/elegant-git/internal/deprecation"
	"github.com/extsoft/elegant-git/internal/doctor"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

func newMigrateCommand() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{
		Use:    "migrate",
		Hidden: true,
		Short:  "Deprecated; migrations run automatically",
		Long:   "Hidden compatibility shim. Remaining Git-install issues are repaired by `eg git doctor`.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			deprecation.Record(deprecation.DEP016, "git migrate", "eg git doctor", "eg git doctor")
			fmt.Fprintln(cmd.ErrOrStderr(), "migrations now run automatically; run `eg git doctor` for the rest")
			if dryRun {
				_, err := doctor.Run(cmd.OutOrStdout(), prompt.NewNonInteractive(), doctor.GitInstall())
				return err
			}
			return doctor.RepairGlobalAliases()
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "report remaining Git-install issues without applying")
	return c
}
