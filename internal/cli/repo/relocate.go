package repo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/memory/repoid"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newRelocateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "relocate <new-path>",
		Short: "Update managed repository path",
		RunE: func(cmd *cobra.Command, args []string) error {
			var newPathArg string
			if err := argspec.ResolveCmd(cmd, args, argspec.Spec{Inputs: []argspec.Input{
				argspec.PositionalInput("new-path", 0, true, "New repository path", &newPathArg, nil),
			}}); err != nil {
				return err
			}
			repoID, err := repoid.ReadLocal()
			if err != nil || repoID == "" {
				return fmt.Errorf("not a configured elegant-git repository")
			}
			newPath, err := filepath.Abs(newPathArg)
			if err != nil {
				return err
			}
			if _, err := os.Stat(newPath); err != nil {
				return fmt.Errorf("path does not exist: %s", newPath)
			}
			s, err := shared.Load()
			if err != nil {
				return err
			}
			if err := shared.RecordPath(s, repoID, newPath); err != nil {
				return err
			}
			if err := shared.Save(s); err != nil {
				return err
			}
			text.InfoText("Repository path updated to " + newPath)
			return nil
		},
	}
}
