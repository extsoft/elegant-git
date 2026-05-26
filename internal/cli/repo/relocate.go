package repo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bees-hive/elegant-git/internal/memory/repoid"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func newRelocateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "relocate <new-path>",
		Short: "Update managed repository path",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			repoID, err := repoid.ReadLocal()
			if err != nil || repoID == "" {
				return fmt.Errorf("not a configured elegant-git repository")
			}
			newPath, err := filepath.Abs(args[0])
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
