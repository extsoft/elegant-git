package legacy

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
)

func TestLegacyShimDoesNotRecurse(t *testing.T) {
	root := &cobra.Command{Use: "git-elegant"}
	var ran bool
	work := &cobra.Command{Use: "work"}
	work.AddCommand(&cobra.Command{
		Use: "save",
		RunE: func(_ *cobra.Command, _ []string) error {
			ran = true
			return nil
		},
	})
	root.AddCommand(work)
	RegisterShims(root)
	root.SetArgs([]string{"save-work"})
	root.SetContext(context.Background())
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("expected work save RunE to run")
	}
}
