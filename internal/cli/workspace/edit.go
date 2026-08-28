package workspace

import (
	"fmt"
	"os"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/cli/completion"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cli/sources"
	memrepo "github.com/bees-hive/elegant-git/internal/memory/repo"
	"github.com/bees-hive/elegant-git/internal/memory/repoid"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

type fieldChange struct {
	label  string
	before string
	after  string
}

type repoTarget struct {
	id   string
	name string
	path string
}

type editPlan struct {
	workspaceID string
	diff        []fieldChange
	planned     shared.Workspace
	apply       []repoTarget
	skip        []repoTarget
	missing     []repoTarget
}

func workspaceEditSpec(name *string) argspec.Spec {
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInputWithComplete("name", 0, true, "Workspace name", name, nil, sources.Workspaces, true),
	}}
}

func newEditCommand() *cobra.Command {
	var workspaceName string
	spec := workspaceEditSpec(&workspaceName)
	c := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a workspace",
		Long:  "Edits workspace fields and optionally applies changes to linked repositories.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
				return err
			}
			s, err := shared.Load()
			if err != nil {
				return err
			}
			id, ws, err := shared.GetWorkspaceByName(s, workspaceName)
			if err != nil {
				return err
			}
			p := prompt.FromContext(cmd.Context())
			plan, err := workspaceEditPlan(cmd, s, id, ws, p)
			if err != nil {
				return err
			}
			if err := workspaceEditSummaryConfirm(plan, p); err != nil {
				return err
			}
			return workspaceEditCommit(cmd, s, plan, p)
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}

func workspaceEditPlan(_ *cobra.Command, s *shared.State, id string, ws *shared.Workspace, p prompt.Prompter) (*editPlan, error) {
	planned := *ws
	var err error
	if !prompt.NonInteractive(p) {
		planned.UserName, err = p.EditOrAccept("Git user.name", ws.UserName)
		if err != nil {
			return nil, err
		}
		planned.UserEmail, err = p.EditOrAccept("Git user.email", ws.UserEmail)
		if err != nil {
			return nil, err
		}
		planned.SigningKey, err = p.Optional("Signing key", suggestField(ws.SigningKey, "user.signingkey"))
		if err != nil {
			return nil, err
		}
		planned.GPGProgram, err = p.Optional("GPG program", suggestField(ws.GPGProgram, "gpg.program"))
		if err != nil {
			return nil, err
		}
		planned.Editor, err = p.Optional("Editor command", suggestField(ws.Editor, "core.editor"))
		if err != nil {
			return nil, err
		}
	}

	plan := &editPlan{
		workspaceID: id,
		planned:     planned,
		diff:        buildFieldDiff(ws, &planned),
	}

	decisions := map[string]bool{}
	remaining := append([]string(nil), ws.LinkedRepos...)
	currentRepoID := ""
	if gitDir, err := memrepo.GitDir(); err == nil {
		if perRepo, err := memrepo.Load(gitDir); err == nil && perRepo.RepoID != "" {
			currentRepoID = perRepo.RepoID
		} else if rid, err := repoid.ReadLocal(); err == nil {
			currentRepoID = rid
		}
	}
	if currentRepoID != "" && containsString(ws.LinkedRepos, currentRepoID) {
		repo, _ := shared.GetRepo(s, currentRepoID)
		repoName := currentRepoID
		if repo != nil {
			repoName = repo.Name
		}
		if !prompt.NonInteractive(p) {
			ok, err := p.Confirm(fmt.Sprintf(`Apply changes to current repository "%s"?`, repoName), true)
			if err != nil {
				return nil, err
			}
			decisions[currentRepoID] = ok
		} else {
			decisions[currentRepoID] = true
		}
		remaining = removeString(remaining, currentRepoID)
	}

	applyAll := prompt.NonInteractive(p)
	skipAll := false
	for _, repoID := range remaining {
		if applyAll {
			decisions[repoID] = true
			continue
		}
		if skipAll {
			decisions[repoID] = false
			continue
		}
		repo, err := shared.GetRepo(s, repoID)
		if err != nil {
			continue
		}
		dec, err := p.BatchChoice(fmt.Sprintf("Apply to %s?", repo.Name), "no")
		if err != nil {
			return nil, err
		}
		switch dec {
		case prompt.BatchConfirm:
			decisions[repoID] = true
		case prompt.BatchApplyAll:
			decisions[repoID] = true
			applyAll = true
		case prompt.BatchSkip:
			decisions[repoID] = false
			skipAll = true
		default:
			decisions[repoID] = false
		}
	}

	for _, repoID := range ws.LinkedRepos {
		repo, err := shared.GetRepo(s, repoID)
		if err != nil {
			continue
		}
		t := repoTarget{id: repoID, name: repo.Name, path: repo.CurrentPath}
		if !decisions[repoID] {
			plan.skip = append(plan.skip, t)
			continue
		}
		if _, err := os.Stat(repo.CurrentPath); err != nil {
			plan.missing = append(plan.missing, t)
			continue
		}
		plan.apply = append(plan.apply, t)
	}
	return plan, nil
}

func buildFieldDiff(before, after *shared.Workspace) []fieldChange {
	show := func(v string) string {
		if v == "" {
			return "(unset)"
		}
		return `"` + v + `"`
	}
	fields := []struct {
		label string
		b, a  string
	}{
		{"user.name", before.UserName, after.UserName},
		{"user.email", before.UserEmail, after.UserEmail},
		{"signing key", before.SigningKey, after.SigningKey},
		{"gpg program", before.GPGProgram, after.GPGProgram},
		{"editor", before.Editor, after.Editor},
	}
	var diff []fieldChange
	for _, f := range fields {
		if f.b == f.a {
			continue
		}
		diff = append(diff, fieldChange{label: f.label, before: show(f.b), after: show(f.a)})
	}
	return diff
}

func workspaceEditSummaryConfirm(plan *editPlan, p prompt.Prompter) error {
	fmt.Println()
	fmt.Printf("Workspace %q changes:\n", plan.planned.Name)
	if len(plan.diff) == 0 {
		fmt.Println("  (no field changes)")
	} else {
		for _, d := range plan.diff {
			fmt.Printf("  %s: %s -> %s\n", d.label, d.before, d.after)
		}
	}
	if len(plan.apply) > 0 {
		fmt.Printf("\nWill apply to %d repository (-ies):\n", len(plan.apply))
		for _, t := range plan.apply {
			fmt.Printf("  - %s    (%s)\n", t.name, t.path)
		}
	}
	if len(plan.skip) > 0 {
		fmt.Printf("\nSkip: %d repository (-ies):\n", len(plan.skip))
		for _, t := range plan.skip {
			fmt.Printf("  - %s    (%s)\n", t.name, t.path)
		}
	}
	if len(plan.missing) > 0 {
		fmt.Printf("\nMissing path: %d repository (-ies); run `git elegant repo configure <name>` from the repository path to fix:\n", len(plan.missing))
		for _, t := range plan.missing {
			fmt.Printf("  - %s    (%s)\n", t.name, t.path)
		}
	}
	if prompt.NonInteractive(p) {
		return nil
	}
	ok, err := p.Confirm("Proceed?", true)
	return errIfNotOK(ok, err)
}

func errIfNotOK(ok bool, err error) error {
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("aborted")
	}
	return nil
}

func workspaceEditCommit(cmd *cobra.Command, s *shared.State, plan *editPlan, p prompt.Prompter) error {
	ws, err := shared.GetWorkspace(s, plan.workspaceID)
	if err != nil {
		return err
	}
	in := shared.UpdateWorkspaceInput{
		UserName:   &plan.planned.UserName,
		UserEmail:  &plan.planned.UserEmail,
		SigningKey: &plan.planned.SigningKey,
		Editor:     &plan.planned.Editor,
		GPGProgram: &plan.planned.GPGProgram,
	}
	if err := shared.UpdateWorkspace(s, plan.workspaceID, in); err != nil {
		return err
	}
	if err := shared.Save(s); err != nil {
		return err
	}
	ws = &plan.planned
	if saved, _ := shared.GetWorkspace(s, plan.workspaceID); saved != nil {
		ws.LinkedRepos = saved.LinkedRepos
	}

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	applied, failed := 0, 0
	force := &shared.Apply{Force: true}
	for _, t := range plan.apply {
		if err := os.Chdir(t.path); err != nil {
			text.ErrorText("repo " + t.name + ": " + err.Error())
			failed++
			continue
		}
		if err := shared.ApplyWorkspace(ws, p, force); err != nil {
			text.ErrorText("repo " + t.name + ": " + err.Error())
			failed++
			continue
		}
		applied++
	}
	_ = os.Chdir(wd)
	text.InfoText(fmt.Sprintf("Applied: %d | Skipped: %d | Failed: %d | Missing: %d",
		applied, len(plan.skip), failed, len(plan.missing)))
	_ = cmd
	return nil
}

func containsString(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func removeString(ss []string, s string) []string {
	var out []string
	for _, x := range ss {
		if x != s {
			out = append(out, x)
		}
	}
	return out
}
