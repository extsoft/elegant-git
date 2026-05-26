package profile

import (
	"fmt"

	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newCreateCommand() *cobra.Command {
	var (
		name       string
		userName   string
		userEmail  string
		signingKey string
		editor     string
		gpgProgram string
	)
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p := prompt.FromContext(cmd.Context())
			var err error
			if name == "" && !prompt.NonInteractive(p) {
				emailHint := suggestField(userEmail, "user.email")
				name, err = p.EditOrAccept("Profile name", defaultProfileName(emailHint))
				if err != nil {
					return err
				}
			}
			name, err = prompt.RequireString(p, name, "profile name")
			if err != nil {
				return err
			}
			userName = suggestField(userName, "user.name")
			userName, err = prompt.Skippable(p, "Git user.name", userName, userName)
			if err != nil {
				return err
			}
			userName, err = prompt.RequireString(p, userName, "user name")
			if err != nil {
				return err
			}
			userEmail = suggestField(userEmail, "user.email")
			userEmail, err = prompt.Skippable(p, "Git user.email", userEmail, userEmail)
			if err != nil {
				return err
			}
			userEmail, err = prompt.RequireString(p, userEmail, "user email")
			if err != nil {
				return err
			}
			signingKey = suggestField(signingKey, "user.signingkey")
			signingKey, err = p.EditOrAccept("Signing key (empty to skip)", signingKey)
			if err != nil {
				return err
			}
			gpgProgram = suggestField(gpgProgram, "gpg.program")
			gpgProgram, err = p.EditOrAccept("GPG program (empty to skip)", gpgProgram)
			if err != nil {
				return err
			}
			editor = suggestField(editor, "core.editor")
			editor, err = p.EditOrAccept("Editor command (empty to skip)", editor)
			if err != nil {
				return err
			}
			s, err := shared.Load()
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
			prof, _ := shared.GetProfile(s, id)
			if err := offerApplyToCurrentRepo(cmd, s, id, prof); err != nil {
				return err
			}
			if err := shared.Save(s); err != nil {
				return err
			}
			text.InfoText(fmt.Sprintf("Created profile %s (%s)", name, id))
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "profile display name")
	c.Flags().StringVar(&userName, "user-name", "", "git user.name")
	c.Flags().StringVar(&userEmail, "user-email", "", "git user.email")
	c.Flags().StringVar(&signingKey, "signing-key", "", "GPG signing key id")
	c.Flags().StringVar(&editor, "editor", "", "core.editor command")
	c.Flags().StringVar(&gpgProgram, "gpg-program", "", "gpg.program path")
	return c
}
