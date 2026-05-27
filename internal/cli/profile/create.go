package profile

import (
	"fmt"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			p := prompt.FromContext(cmd.Context())
			spec := argspec.Spec{Inputs: []argspec.Input{
				argspec.FlagInput("name", "name", true, "Profile name", &name, func() string {
					return defaultProfileName(suggestField(userEmail, "user.email"))
				}),
				argspec.FlagInput("user-name", "user-name", true, "Git user.name", &userName, func() string {
					return suggestField(userName, "user.name")
				}),
				argspec.FlagInput("user-email", "user-email", true, "Git user.email", &userEmail, func() string {
					return suggestField(userEmail, "user.email")
				}),
				argspec.FlagInput("signing-key", "signing-key", false, "Signing key (empty to skip)", &signingKey, func() string {
					return suggestField(signingKey, "user.signingkey")
				}),
				argspec.FlagInput("gpg-program", "gpg-program", false, "GPG program (empty to skip)", &gpgProgram, func() string {
					return suggestField(gpgProgram, "gpg.program")
				}),
				argspec.FlagInput("editor", "editor", false, "Editor command (empty to skip)", &editor, func() string {
					return suggestField(editor, "core.editor")
				}),
			}}
			if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
				return err
			}
			if name == "" {
				return fmt.Errorf("profile name is required")
			}
			if userName == "" || userEmail == "" {
				return fmt.Errorf("user name and email are required")
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
			_ = p
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
