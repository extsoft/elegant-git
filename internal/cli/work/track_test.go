package work

import (
	"context"
	"testing"

	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/prompt"
)

func TestRemoteBranchesMatching(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/rremote\norigin/master\nother-upstream/barr"
	git.Use(m)

	got := remoteBranchesMatching("rremote")
	if len(got) != 1 || got[0] != "origin/rremote" {
		t.Fatalf("got %+v", got)
	}
}

func TestTrackLogicPromptsAfterFetch(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["for-each-ref --format=%(refname:short)\t%(objectname:short) refs/remotes"] = "origin/OPS-2114\tabc"
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/OPS-2114"
	git.Use(m)

	p := &trackRecordingPrompter{
		pickValues: []string{"origin/OPS-2114"},
		editValues: []string{""},
	}
	if err := trackLogic(context.Background(), p, "", ""); err != nil {
		t.Fatal(err)
	}
	if len(m.Calls) < 2 {
		t.Fatalf("calls=%v", m.Calls)
	}
	if m.Calls[0].Args[0] != "fetch" {
		t.Fatalf("first call=%v", m.Calls[0].Args)
	}
	fetchIdx := -1
	for i, c := range m.Calls {
		if len(c.Args) > 0 && c.Args[0] == "fetch" {
			fetchIdx = i
			break
		}
	}
	if fetchIdx < 0 {
		t.Fatal("missing fetch")
	}
	if len(p.edits) != 1 || p.edits[0].suggested != "OPS-2114" {
		t.Fatalf("edits=%+v", p.edits)
	}
	checkoutIdx := -1
	for i, c := range m.Calls {
		if len(c.Args) > 0 && c.Args[0] == "checkout" {
			checkoutIdx = i
		}
	}
	if checkoutIdx <= fetchIdx {
		t.Fatalf("fetch=%d checkout=%d", fetchIdx, checkoutIdx)
	}
	if len(p.picks) != 1 {
		t.Fatalf("picks=%v", p.picks)
	}
}

func TestTrackLogicLocalBranchOverride(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/feature/133"
	git.Use(m)

	p := &trackRecordingPrompter{editValues: []string{"task-133"}}
	if err := trackLogic(context.Background(), p, "origin/feature/133", ""); err != nil {
		t.Fatal(err)
	}
	if len(p.picks) != 0 {
		t.Fatalf("unexpected pick")
	}
	if len(p.edits) != 1 || p.edits[0].suggested != "feature/133" {
		t.Fatalf("edits=%+v", p.edits)
	}
	last := m.Calls[len(m.Calls)-1]
	if last.Args[0] != "checkout" || last.Args[2] != "task-133" {
		t.Fatalf("checkout=%v", last.Args)
	}
}

func TestTrackLogicNonInteractiveSkipsPrompts(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/feature/133"
	git.Use(m)

	p := prompt.NewNonInteractive()
	if err := trackLogic(context.Background(), p, "origin/feature/133", ""); err != nil {
		t.Fatal(err)
	}
	last := m.Calls[len(m.Calls)-1]
	if last.Args[0] != "checkout" || last.Args[2] != "feature/133" {
		t.Fatalf("checkout=%v", last.Args)
	}
}

type trackRecordingPrompter struct {
	pickValues []string
	editValues []string
	picks      []string
	edits      []struct{ label, suggested string }
	pickIdx    int
	editIdx    int
}

func (p *trackRecordingPrompter) String(string, string) (string, error) { return "", nil }
func (p *trackRecordingPrompter) Confirm(string) (bool, error)          { return false, nil }
func (p *trackRecordingPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}
func (p *trackRecordingPrompter) Required(string, string) error { return nil }
func (p *trackRecordingPrompter) BatchChoice(string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}

func (p *trackRecordingPrompter) Pick(label string, _ []prompt.Choice) (string, error) {
	p.picks = append(p.picks, label)
	if p.pickIdx < len(p.pickValues) {
		v := p.pickValues[p.pickIdx]
		p.pickIdx++
		return v, nil
	}
	return "", nil
}

func (p *trackRecordingPrompter) EditOrAccept(label, suggested string) (string, error) {
	p.edits = append(p.edits, struct{ label, suggested string }{label, suggested})
	if p.editIdx < len(p.editValues) {
		v := p.editValues[p.editIdx]
		p.editIdx++
		if v != "" {
			return v, nil
		}
	}
	return suggested, nil
}
