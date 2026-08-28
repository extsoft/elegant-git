package workspace

import (
	"fmt"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/cli/completion"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func createSpec(name, userName, userEmail, signingKey, gpgProgram, editor *string) argspec.Spec {
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInput("name", 0, true, "Workspace name", name, func() string {
			return defaultWorkspaceName(suggestField(*userEmail, "user.email"))
		}),
		argspec.PositionalInput("user-name", 1, true, "Git user.name", userName, func() string {
			return suggestField(*userName, "user.name")
		}),
		argspec.PositionalInput("user-email", 2, true, "Git user.email", userEmail, func() string {
			return suggestField(*userEmail, "user.email")
		}),
		argspec.PositionalInput("signing-key", 3, false, "Signing key", signingKey, func() string {
			return suggestField(*signingKey, "user.signingkey")
		}),
		argspec.PositionalInput("gpg-program", 4, false, "GPG program", gpgProgram, func() string {
			return suggestField(*gpgProgram, "gpg.program")
		}),
		argspec.PositionalInput("editor", 5, false, "Editor command", editor, func() string {
			return suggestField(*editor, "core.editor")
		}),
	}}
}

func newCreateCommand() *cobra.Command {
	var (
		name       string
		userName   string
		userEmail  string
		signingKey string
		editor     string
		gpgProgram string
	)
	spec := createSpec(&name, &userName, &userEmail, &signingKey, &gpgProgram, &editor)
	c := &cobra.Command{
		Use:   "create <name> <user-name> <user-email> [<signing-key>] [<gpg-program>] [<editor>]",
		Short: "Create a workspace",
		Long:  "Creates a Git workspace in shared memory. Required fields can be passed as arguments or prompted interactively.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
				return err
			}
			if name == "" {
				return fmt.Errorf("workspace name is required")
			}
			if userName == "" || userEmail == "" {
				return fmt.Errorf("user name and email are required")
			}
			s, err := shared.Load()
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
			ws, _ := shared.GetWorkspace(s, id)
			if err := offerApplyToCurrentRepo(cmd, s, id, ws); err != nil {
				return err
			}
			if err := shared.Save(s); err != nil {
				return err
			}
			text.InfoText(fmt.Sprintf("Created workspace %s (%s)", name, id))
			return nil
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}
