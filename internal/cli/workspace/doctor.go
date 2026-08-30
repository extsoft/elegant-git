package workspace

import (
	"fmt"
	"sort"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	"github.com/extsoft/elegant-git/internal/cli/completion"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	"github.com/extsoft/elegant-git/internal/doctor"
	"github.com/extsoft/elegant-git/internal/memory/shared"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func doctorSpec(name *string) argspec.Spec {
	in := argspec.PositionalInputWithComplete("name", 0, false, "Workspace name", name, nil, sources.Workspaces, true)
	in.OmitInteractive = true
	return argspec.Spec{Inputs: []argspec.Input{in}}
}

func newDoctorCommand() *cobra.Command {
	var name string
	spec := doctorSpec(&name)
	c := &cobra.Command{
		Use:   "doctor [name]",
		Short: "Diagnose and repair a workspace",
		Long:  "Checks one workspace and its linked repositories, then offers repairs. Interactive mode confirms each repair; non-interactive mode only reports. When name is omitted, uses the linked workspace or asks.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
				return err
			}
			return doctorRun(cmd, name)
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func doctorRun(cmd *cobra.Command, name string) error {
	w := cmd.OutOrStdout()
	p := prompt.FromContext(cmd.Context())
	s, err := shared.Load()
	if err != nil {
		return err
	}
	id, ws, err := resolveDoctorWorkspace(s, name, p)
	if err != nil {
		return err
	}

	findings := doctor.Workspace(s, id, p)
	findings = append(findings, doctor.LinkedRepos(s, id, p)...)
	if len(findings) == 0 {
		fmt.Fprintln(w, "Workspace "+ws.Name+" looks healthy")
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
	if err := shared.Validate(s); err != nil {
		return err
	}
	if err := shared.Save(s); err != nil {
		return err
	}
	text.InfoText("Repaired workspace " + ws.Name)
	return nil
}

func resolveDoctorWorkspace(s *shared.State, name string, p prompt.Prompter) (string, *shared.Workspace, error) {
	id, ws, err := resolveNamedOrLinkedWorkspace(s, name)
	if err == nil {
		return id, ws, nil
	}
	if name != "" {
		return "", nil, err
	}
	if prompt.NonInteractive(p) {
		return "", nil, fmt.Errorf("workspace name is required")
	}
	return pickDoctorWorkspace(s, p)
}

func pickDoctorWorkspace(s *shared.State, p prompt.Prompter) (string, *shared.Workspace, error) {
	choices := make([]prompt.Choice, 0, len(s.Workspaces))
	for _, ws := range shared.ListWorkspaces(s) {
		if ws == nil {
			continue
		}
		choices = append(choices, prompt.Choice{
			Value:       ws.Name,
			Description: ws.UserName + " <" + ws.UserEmail + ">",
		})
	}
	sort.Slice(choices, func(i, j int) bool { return choices[i].Value < choices[j].Value })
	if len(choices) == 0 {
		return "", nil, fmt.Errorf("no workspaces to examine; create one with workspace new")
	}
	picked, err := p.Pick("Workspace", choices, "")
	if err != nil {
		if err == prompt.ErrUserCancelled {
			return "", nil, fmt.Errorf("workspace name is required")
		}
		return "", nil, err
	}
	return shared.GetWorkspaceByName(s, picked)
}
