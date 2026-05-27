package memory

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/spf13/cobra"
)

func newProfilesCommand() *cobra.Command {
	var format string
	c := &cobra.Command{
		Use:   "profiles [name]",
		Short: "List profiles or show one profile's details",
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if err := argspec.ResolveCmd(cmd, args, argspec.Spec{Inputs: []argspec.Input{
				argspec.PositionalInput("name", 0, false, "Profile name", &name, nil),
			}}); err != nil {
				return err
			}
			s, err := shared.Load()
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if name != "" {
				return showProfile(w, s, name, format)
			}
			return listProfiles(w, s, format)
		},
	}
	c.Flags().StringVar(&format, "format", "table", "output format: table or json")
	return c
}

func listProfiles(w io.Writer, s *shared.State, format string) error {
	if format == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(shared.ListProfiles(s))
	}
	type row struct {
		id string
		p  *shared.Profile
	}
	var rows []row
	for id, p := range shared.ListProfiles(s) {
		if p != nil {
			rows = append(rows, row{id, p})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].p.Name < rows[j].p.Name })
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s <%s>\t%d repo(s)\n", r.p.Name, r.p.UserName, r.p.UserEmail, len(r.p.LinkedRepos))
	}
	return nil
}

func showProfile(w io.Writer, s *shared.State, name, format string) error {
	id, p, err := shared.GetProfileByName(s, name)
	if err != nil {
		return err
	}
	if format == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"id":      id,
			"profile": p,
		})
	}
	printProfileFields(w, "", id, p)
	printLinkedRepos(w, s, p, "")
	return nil
}
