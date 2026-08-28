package git

import (
	"strings"

	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func offerCreateWorkspaceFromGlobal(cmd *cobra.Command) error {
	p := prompt.FromContext(cmd.Context())
	userName := git.ConfigGlobalGet("user.name")
	userEmail := git.ConfigGlobalGet("user.email")
	signingKey := git.ConfigGlobalGet("user.signingkey")
	editor := git.ConfigGlobalGet("core.editor")
	gpgProgram := git.ConfigGlobalGet("gpg.program")
	if userName == "" || userEmail == "" {
		return nil
	}
	s, err := shared.Load()
	if err != nil {
		return err
	}
	if _, existing, ok := shared.FindWorkspaceByFields(s, userName, userEmail, signingKey, editor, gpgProgram); ok {
		text.InfoText("Workspace already exists: " + existing.Name)
		return nil
	}
	if prompt.NonInteractive(p) {
		return nil
	}
	ok, err := p.Confirm("Create a reusable workspace from these global values?", false)
	if err != nil || !ok {
		return err
	}
	name, err := p.EditOrAccept("Workspace name", defaultWorkspaceNameFromEmail(userEmail))
	if err != nil {
		return err
	}
	id, err := shared.CreateWorkspace(s, shared.CreateWorkspaceInput{
		Name: name, UserName: userName, UserEmail: userEmail,
		SigningKey: signingKey, Editor: editor, GPGProgram: gpgProgram,
	})
	if err != nil {
		return err
	}
	if err := shared.Save(s); err != nil {
		return err
	}
	text.InfoText("Created workspace " + name + " (" + id + ")")
	return nil
}

func defaultWorkspaceNameFromEmail(email string) string {
	if i := strings.Index(email, "@"); i > 0 {
		return email[:i]
	}
	return email
}
