package profile

import (
	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/cli/completion"
	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/cli/sources"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func deleteSpec(name *string) argspec.Spec {
	return argspec.Spec{Inputs: []argspec.Input{
		argspec.PositionalInputWithComplete("name", 0, true, "Profile name", name, nil, sources.Profiles, true),
	}}
}

func newDeleteCommand() *cobra.Command {
	var name string
	spec := deleteSpec(&name)
	c := &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a profile",
		Long:  "Deletes a profile from shared memory when no repositories are linked to it. When name is omitted, choose from existing profiles.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := argspec.ResolveCmd(cmd, args, spec); err != nil {
				return err
			}
			s, err := shared.Load()
			if err != nil {
				return err
			}
			id, _, err := shared.GetProfileByName(s, name)
			if err != nil {
				return err
			}
			if err := shared.DeleteProfile(s, id); err != nil {
				return err
			}
			if err := shared.Save(s); err != nil {
				return err
			}
			text.InfoText("Deleted profile " + name)
			return nil
		},
	}
	c.SetHelpFunc(cliruntime.CommandHelp)
	completion.Attach(c, spec)
	return c
}
