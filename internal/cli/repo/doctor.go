package repo

import (
	"fmt"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/doctor"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newDoctorCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose and repair the current repository",
		Long:  "Finds registry, identity, and legacy configuration problems for the repository at the current working directory and suggests a repair for each. Interactive mode confirms yes/no repairs; non-interactive mode only reports and exits non-zero when issues remain.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return doctorRun(cmd)
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	return c
}

func doctorRun(cmd *cobra.Command) error {
	w := cmd.OutOrStdout()
	p := prompt.FromContext(cmd.Context())
	s, err := shared.Load()
	if err != nil {
		return err
	}
	findings, err := doctor.CurrentRepo(s, p)
	if err != nil {
		return err
	}
	if len(findings) == 0 {
		fmt.Fprintln(w, "Repository looks healthy")
		return nil
	}
	changed, err := doctor.Run(w, p, findings)
	if err != nil {
		return err
	}
	if prompt.NonInteractive(p) {
		return fmt.Errorf("%d issue(s) found; run interactively to repair", len(findings))
	}
	if !changed {
		return nil
	}
	if err := shared.Validate(s); err != nil {
		return err
	}
	if err := shared.Save(s); err != nil {
		return err
	}
	text.InfoText("Repaired current repository")
	return nil
}
