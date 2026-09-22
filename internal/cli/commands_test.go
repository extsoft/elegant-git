package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	clicatalog "github.com/extsoft/elegant-git/internal/cli/catalog"
)

func findGroupActions(object string) []string {
	for _, g := range clicatalog.Groups {
		if g.Object == object {
			var actions []string
			for _, c := range g.Commands {
				actions = append(actions, c.Action)
			}
			return actions
		}
	}
	return nil
}

func hasAction(actions []string, action string) bool {
	return slices.Contains(actions, action)
}

func TestSelfGroupActions(t *testing.T) {
	got := findGroupActions("self")
	want := []string{"configure", "list", "doctor"}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestListActionsInGroups(t *testing.T) {
	for _, tc := range []struct {
		object string
		has    string
		lacks  string
	}{
		{"self", "list", "status"},
		{"workspace", "list", "status"},
		{"repo", "list", "status"},
		{"hook", "list", "status"},
		{"work", "list", "status"},
	} {
		actions := findGroupActions(tc.object)
		if !hasAction(actions, tc.has) {
			t.Errorf("%s missing %q in %v", tc.object, tc.has, actions)
		}
		if tc.lacks != "" && hasAction(actions, tc.lacks) {
			t.Errorf("%s should not have %q", tc.object, tc.lacks)
		}
	}
}

func TestWriteRootUsageActionOnlyAligned(t *testing.T) {
	var buf bytes.Buffer
	clicatalog.WriteRootUsage(&buf)
	text := buf.String()

	assertAlignedSection(t, text, "    -")
	var allRows []string
	for _, g := range clicatalog.Groups {
		rows := sectionRows(text, "  "+g.Object+" —", "    ")
		if len(rows) != len(g.Commands) {
			t.Errorf("%s: got %d action rows, want %d", g.Object, len(rows), len(g.Commands))
		}
		for i, row := range rows {
			name := firstField(strings.TrimSpace(row))
			if i < len(g.Commands) && name != g.Commands[i].Action {
				t.Errorf("%s: row %q name %q, want action %q", g.Object, row, name, g.Commands[i].Action)
			}
			if strings.HasPrefix(strings.TrimSpace(row), g.Object+" ") {
				t.Errorf("action row prefixes object: %q", row)
			}
		}
		allRows = append(allRows, rows...)
	}
	assertSameDescColumn(t, "all commands", allRows)
}

func TestWriteObjectUsageAligned(t *testing.T) {
	var buf bytes.Buffer
	clicatalog.WriteObjectUsage(&buf, "self")
	text := buf.String()

	if strings.Contains(text, "Objects:") {
		t.Fatal("object help should not show root catalog")
	}
	assertAlignedSection(t, text, "    -")
	rows := sectionRows(text, "Actions:", "  ")
	assertSameDescColumn(t, "self", rows)
	actions := findGroupActions("self")
	if len(rows) != len(actions) {
		t.Fatalf("got %d action rows, want %d", len(rows), len(actions))
	}
	for i, row := range rows {
		name := firstField(strings.TrimSpace(row))
		if name != actions[i] {
			t.Errorf("row %q name %q, want %q", row, name, actions[i])
		}
	}
}

func sectionRows(text, heading, indent string) []string {
	lines := strings.Split(text, "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(line, heading) {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return nil
	}
	var rows []string
	for _, line := range lines[start:] {
		if !strings.HasPrefix(line, indent) {
			break
		}
		rest := line[len(indent):]
		if rest == "" || rest[0] == ' ' {
			break
		}
		rows = append(rows, line)
	}
	return rows
}

func firstField(s string) string {
	if i := strings.IndexByte(s, ' '); i >= 0 {
		return s[:i]
	}
	return s
}

func assertAlignedSection(t *testing.T, text, prefix string) {
	t.Helper()
	var rows []string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, prefix) {
			rows = append(rows, line)
		}
	}
	if len(rows) == 0 {
		t.Fatalf("no rows with prefix %q", prefix)
	}
	assertSameDescColumn(t, prefix, rows)
}

func assertSameDescColumn(t *testing.T, label string, rows []string) {
	t.Helper()
	if len(rows) == 0 {
		t.Fatalf("%s: no rows", label)
	}
	col := descriptionStart(rows[0])
	if col < 0 {
		t.Fatalf("%s: no description on %q", label, rows[0])
	}
	for _, row := range rows[1:] {
		got := descriptionStart(row)
		if got != col {
			t.Errorf("%s: description column %d on %q, want %d", label, got, row, col)
		}
	}
}

func descriptionStart(line string) int {
	i := 0
	for i < len(line) && line[i] == ' ' {
		i++
	}
	j := strings.Index(line[i:], "  ")
	if j < 0 {
		return -1
	}
	k := i + j
	for k < len(line) && line[k] == ' ' {
		k++
	}
	if k >= len(line) {
		return -1
	}
	return k
}
