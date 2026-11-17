package work

import (
	"testing"

	"github.com/extsoft/elegant-git/internal/cli/catalog"
)

func TestDirtyClassCatalog(t *testing.T) {
	want := map[string]string{
		"self.configure":   "tolerate",
		"self.list":        "tolerate",
		"self.doctor":      "tolerate",
		"workspace.list":   "tolerate",
		"workspace.new":    "tolerate",
		"workspace.link":   "tolerate",
		"workspace.edit":   "tolerate",
		"workspace.delete": "tolerate",
		"workspace.fetch":  "tolerate",
		"workspace.exec":   "tolerate",
		"workspace.doctor": "tolerate",
		"repo.list":        "tolerate",
		"repo.init":        "create",
		"repo.clone":       "create",
		"repo.configure":   "tolerate",
		"repo.sync":        "tolerate",
		"repo.prune":       "prepare-legacy",
		"repo.doctor":      "tolerate",
		"hook.list":        "tolerate",
		"hook.new":         "tolerate",
		"hook.edit":        "tolerate",
		"work.start":       "prepare",
		"work.save":        "consume",
		"work.list":        "tolerate",
		"work.polish":      "prepare-legacy",
		"work.sync":        "prepare-legacy",
		"work.push":        "prepare-legacy",
		"work.switch":      "prepare",
		"work.accept":      "prepare-legacy",
		"release.new":      "prepare-legacy",
		"release.notes":    "tolerate",
	}
	seen := 0
	for _, g := range catalog.Groups {
		for _, c := range g.Commands {
			seen++
			id := g.Object + "." + c.Action
			if want[id] == "" {
				t.Fatalf("unclassified %s", id)
			}
			if got := dirtyClass(g.Object, c.Action); got != want[id] {
				t.Fatalf("%s class %s want %s", id, got, want[id])
			}
		}
	}
	if seen != len(want) {
		t.Fatalf("catalog %d want %d", seen, len(want))
	}
	if dirtyClass("version", "") != "tolerate" || dirtyClass("completion", "bash") != "tolerate" {
		t.Fatal("read-only commands")
	}
}
