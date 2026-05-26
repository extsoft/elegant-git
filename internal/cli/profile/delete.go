package profile

import (
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newDeleteCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := shared.Load()
			if err != nil {
				return err
			}
			id, _, err := shared.GetProfileByName(s, args[0])
			if err != nil {
				return err
			}
			if err := shared.DeleteProfile(s, id); err != nil {
				return err
			}
			if err := shared.Save(s); err != nil {
				return err
			}
			text.InfoText("Deleted profile " + args[0])
			return nil
		},
	}
	return c
}
