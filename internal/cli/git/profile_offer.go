package git

import (
	"strings"

	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func offerCreateProfileFromGlobal(cmd *cobra.Command) error {
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
	if _, existing, ok := shared.FindProfileByFields(s, userName, userEmail, signingKey, editor, gpgProgram); ok {
		text.InfoText("Profile already exists: " + existing.Name)
		return nil
	}
	if prompt.NonInteractive(p) {
		return nil
	}
	ok, err := p.Confirm("Create a reusable profile from these global values?")
	if err != nil || !ok {
		return err
	}
	name, err := p.EditOrAccept("Profile name", defaultProfileNameFromEmail(userEmail))
	if err != nil {
		return err
	}
	id, err := shared.CreateProfile(s, shared.CreateProfileInput{
		Name: name, UserName: userName, UserEmail: userEmail,
		SigningKey: signingKey, Editor: editor, GPGProgram: gpgProgram,
	})
	if err != nil {
		return err
	}
	if err := shared.Save(s); err != nil {
		return err
	}
	text.InfoText("Created profile " + name + " (" + id + ")")
	return nil
}

func defaultProfileNameFromEmail(email string) string {
	if i := strings.Index(email, "@"); i > 0 {
		return email[:i]
	}
	return email
}
