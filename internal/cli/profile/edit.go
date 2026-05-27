package profile

import (
	"fmt"
	"os"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
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
	profileID string
	diff      []fieldChange
	planned   shared.Profile
	apply     []repoTarget
	skip      []repoTarget
	missing   []repoTarget
}

func newEditCommand() *cobra.Command {
	var (
		userName   string
		userEmail  string
		signingKey string
		editor     string
		gpgProgram string
	)
	c := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			var profileName string
			if err := argspec.ResolveCmd(cmd, args, argspec.Spec{Inputs: []argspec.Input{
				argspec.PositionalInput("name", 0, true, "Profile name", &profileName, nil),
			}}); err != nil {
				return err
			}
			s, err := shared.Load()
			if err != nil {
				return err
			}
			id, prof, err := shared.GetProfileByName(s, profileName)
			if err != nil {
				return err
			}
			p := prompt.FromContext(cmd.Context())
			plan, err := profileEditPlan(cmd, s, id, prof, p, userName, userEmail, signingKey, editor, gpgProgram)
			if err != nil {
				return err
			}
			if err := profileEditSummaryConfirm(plan, p); err != nil {
				return err
			}
			return profileEditCommit(cmd, s, plan, p)
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	c.Flags().StringVar(&userName, "user-name", "", "git user.name")
	c.Flags().StringVar(&userEmail, "user-email", "", "git user.email")
	c.Flags().StringVar(&signingKey, "signing-key", "", "GPG signing key id")
	c.Flags().StringVar(&editor, "editor", "", "core.editor command")
	c.Flags().StringVar(&gpgProgram, "gpg-program", "", "gpg.program path")
	return c
}

func profileEditPlan(_ *cobra.Command, s *shared.State, id string, prof *shared.Profile, p prompt.Prompter, userName, userEmail, signingKey, editor, gpgProgram string) (*editPlan, error) {
	planned := *prof
	var err error
	if userName != "" {
		planned.UserName = userName
	} else if !prompt.NonInteractive(p) {
		planned.UserName, err = p.EditOrAccept("Git user.name", prof.UserName)
		if err != nil {
			return nil, err
		}
	}
	if userEmail != "" {
		planned.UserEmail = userEmail
	} else if !prompt.NonInteractive(p) {
		planned.UserEmail, err = p.EditOrAccept("Git user.email", prof.UserEmail)
		if err != nil {
			return nil, err
		}
	}
	if signingKey != "" {
		planned.SigningKey = signingKey
	} else if !prompt.NonInteractive(p) {
		planned.SigningKey, err = p.EditOrAccept("Signing key (empty to clear)", prof.SigningKey)
		if err != nil {
			return nil, err
		}
	}
	if gpgProgram != "" {
		planned.GPGProgram = gpgProgram
	} else if !prompt.NonInteractive(p) {
		planned.GPGProgram, err = p.EditOrAccept("GPG program (empty to clear)", prof.GPGProgram)
		if err != nil {
			return nil, err
		}
	}
	if editor != "" {
		planned.Editor = editor
	} else if !prompt.NonInteractive(p) {
		planned.Editor, err = p.EditOrAccept("Editor command (empty to clear)", prof.Editor)
		if err != nil {
			return nil, err
		}
	}

	plan := &editPlan{
		profileID: id,
		planned:   planned,
		diff:      buildFieldDiff(prof, &planned),
	}

	decisions := map[string]bool{}
	remaining := append([]string(nil), prof.LinkedRepos...)
	currentRepoID := ""
	if gitDir, err := memrepo.GitDir(); err == nil {
		if perRepo, err := memrepo.Load(gitDir); err == nil && perRepo.RepoID != "" {
			currentRepoID = perRepo.RepoID
		} else if rid, err := repoid.ReadLocal(); err == nil {
			currentRepoID = rid
		}
	}
	if currentRepoID != "" && containsString(prof.LinkedRepos, currentRepoID) {
		repo, _ := shared.GetRepo(s, currentRepoID)
		repoName := currentRepoID
		if repo != nil {
			repoName = repo.Name
		}
		if !prompt.NonInteractive(p) {
			ok, err := p.Confirm(fmt.Sprintf(`Apply changes to current repository "%s"?`, repoName))
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
		dec, err := p.BatchChoice(fmt.Sprintf("Apply to %s?", repo.Name))
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

	for _, repoID := range prof.LinkedRepos {
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

func buildFieldDiff(before, after *shared.Profile) []fieldChange {
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

func profileEditSummaryConfirm(plan *editPlan, p prompt.Prompter) error {
	fmt.Println()
	fmt.Printf("Profile %q changes:\n", plan.planned.Name)
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
		fmt.Printf("\nMissing path: %d repository (-ies); run `git elegant repo relocate` to fix:\n", len(plan.missing))
		for _, t := range plan.missing {
			fmt.Printf("  - %s    (%s)\n", t.name, t.path)
		}
	}
	if prompt.NonInteractive(p) {
		return nil
	}
	ok, err := p.Confirm("Proceed?")
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

func profileEditCommit(cmd *cobra.Command, s *shared.State, plan *editPlan, p prompt.Prompter) error {
	prof, err := shared.GetProfile(s, plan.profileID)
	if err != nil {
		return err
	}
	in := shared.UpdateProfileInput{
		UserName:   &plan.planned.UserName,
		UserEmail:  &plan.planned.UserEmail,
		SigningKey: &plan.planned.SigningKey,
		Editor:     &plan.planned.Editor,
		GPGProgram: &plan.planned.GPGProgram,
	}
	if err := shared.UpdateProfile(s, plan.profileID, in); err != nil {
		return err
	}
	if err := shared.Save(s); err != nil {
		return err
	}
	prof = &plan.planned
	if saved, _ := shared.GetProfile(s, plan.profileID); saved != nil {
		prof.LinkedRepos = saved.LinkedRepos
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
		if err := shared.ApplyProfile(prof, p, force); err != nil {
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
