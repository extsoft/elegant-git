package legacy

import (
	"fmt"
	"os"
	"strings"

	"github.com/bees-hive/elegant-git/internal/deprecation"
	"github.com/spf13/cobra"
)

// RegisterShims adds hidden legacy flat-name commands that delegate to new paths.
func RegisterShims(root *cobra.Command) {
	for legacyName, path := range LegacyToPath {
		name := legacyName
		targetPath := append([]string(nil), path...)
		root.AddCommand(&cobra.Command{
			Use:    name,
			Hidden: true,
			RunE: func(cmd *cobra.Command, argv []string) error {
				deprecation.RecordLegacyCommand(name, ReplacementCommand(name))
				return runTarget(root, cmd, targetPath, argv)
			},
		})
	}
	root.AddCommand(&cobra.Command{
		Use:    "show-commands",
		Hidden: true,
		Run: func(_ *cobra.Command, _ []string) {
			deprecation.RecordShowCommands()
			for _, n := range LegacyNames() {
				if n == "show-commands" {
					continue
				}
				fmt.Fprintln(os.Stdout, n)
			}
		},
	})
}

func runTarget(root, shim *cobra.Command, path, argv []string) error {
	target, _, err := root.Find(path)
	if err != nil {
		return err
	}
	target.SetContext(shim.Context())
	target.SetArgs(argv)
	target.SetOut(shim.OutOrStdout())
	target.SetErr(shim.OutOrStderr())
	target.SetIn(shim.InOrStdin())
	if target.RunE != nil {
		return target.RunE(target, argv)
	}
	if target.Run != nil {
		target.Run(target, argv)
		return nil
	}
	return target.Help()
}

// NewArgPath joins legacy path for delegation.
func NewArgPath(legacy string) string {
	p, ok := LegacyToPath[legacy]
	if !ok {
		return ""
	}
	return strings.Join(p, " ")
}
