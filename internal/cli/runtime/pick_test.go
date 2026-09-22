package runtime

import (
	"bytes"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/cli/catalog"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/spf13/cobra"
)

func TestPickActionOrHelpThenQuit(t *testing.T) {
	var buf bytes.Buffer
	c := &cobra.Command{Use: "work"}
	catalog.AttachObjectHelp(c, "work")
	c.SetOut(&buf)
	p := &pickPrompter{picks: []string{"help", "quit"}}
	ans, err := PickActionOrHelp(c, p, "What now", nil, "quit")
	if err != nil {
		t.Fatal(err)
	}
	if ans != "quit" {
		t.Fatalf("ans=%q", ans)
	}
	if len(p.asked) != 2 {
		t.Fatalf("asked=%v", p.asked)
	}
	if !strings.Contains(buf.String(), "work — day-to-day contributions") {
		t.Fatalf("usage=%q", buf.String())
	}
}

type pickPrompter struct {
	picks []string
	asked []string
	idx   int
}

func (p *pickPrompter) String(string, string) (string, error) { return "", nil }
func (p *pickPrompter) Confirm(string, bool) (bool, error)    { return false, nil }
func (p *pickPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}
func (p *pickPrompter) Pick(label string, _ []prompt.Choice, _ string) (string, error) {
	p.asked = append(p.asked, label)
	if p.idx < len(p.picks) {
		v := p.picks[p.idx]
		p.idx++
		return v, nil
	}
	return "quit", nil
}
func (p *pickPrompter) Required(string, string) error               { return nil }
func (p *pickPrompter) EditOrAccept(string, string) (string, error) { return "", nil }
func (p *pickPrompter) Optional(string, string) (string, error)     { return "", nil }
func (p *pickPrompter) Closed(string, []string, string, bool) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *pickPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}
