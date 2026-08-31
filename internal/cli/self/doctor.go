package self

import (
	"fmt"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/doctor"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newDoctorCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose and repair your Git installation",
		Long:  "Checks the Git installation and offers repairs. Interactive mode confirms each repair; non-interactive mode only reports.",
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
	findings := doctor.GitInstall()
	if len(findings) == 0 {
		fmt.Fprintln(w, "Git installation looks healthy")
		return nil
	}
	changed, err := doctor.Run(w, p, findings)
	if err != nil {
		return err
	}
	if prompt.NonInteractive(p) {
		if n := doctor.RepairableCount(findings); n > 0 {
			return fmt.Errorf("%d issue(s) found; run interactively to repair", n)
		}
		return nil
	}
	if !changed {
		return nil
	}
	text.InfoText("Repaired Git installation")
	return nil
}
