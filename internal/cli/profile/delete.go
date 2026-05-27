package profile

import (
	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newDeleteCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if err := argspec.ResolveCmd(cmd, args, argspec.Spec{Inputs: []argspec.Input{
				argspec.PositionalInput("name", 0, true, "Profile name", &name, nil),
			}}); err != nil {
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
	return c
}
