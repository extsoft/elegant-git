package repo

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/cli/catalog"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func TestRunBareNonInteractive(t *testing.T) {
	var buf bytes.Buffer
	c := NewCommand()
	catalog.AttachObjectHelp(c, "repo")
	c.SetOut(&buf)
	c.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	if err := runBare(c, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "repo — manage repositories") {
		t.Fatalf("usage=%q", buf.String())
	}
}

func TestRunSessionAsksQuit(t *testing.T) {
	var ran []string
	orig := dispatchAction
	dispatchAction = func(_ *cobra.Command, action string, _ ...string) error {
		ran = append(ran, action)
		return nil
	}
	defer func() { dispatchAction = orig }()

	var buf bytes.Buffer
	text.SetOutput(&buf)
	defer text.SetOutput(os.Stdout)

	p := &sessionPrompter{picks: []string{"quit"}}
	c := NewCommand()
	c.SetContext(prompt.WithPrompter(context.Background(), p))
	err := runSession(c, func() snapshot {
		return snapshot{InGit: true, Configured: true}
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ran) != 0 {
		t.Fatalf("ran %v", ran)
	}
	if len(p.asked) != 1 || p.asked[0] != "What now" {
		t.Fatalf("asked=%v", p.asked)
	}
	out := buf.String()
	if !strings.Contains(out, "==>> Detection action...") || !strings.Contains(out, "selected: ask") {
		t.Fatalf("eval=%q", out)
	}
	if !strings.Contains(out, "configured? yes") {
		t.Fatalf("eval=%q", out)
	}
}

func TestRunSessionDispatchList(t *testing.T) {
	var ran []string
	orig := dispatchAction
	dispatchAction = func(_ *cobra.Command, action string, _ ...string) error {
		ran = append(ran, action)
		return nil
	}
	defer func() { dispatchAction = orig }()

	p := &sessionPrompter{picks: []string{"list"}}
	c := NewCommand()
	c.SetContext(prompt.WithPrompter(context.Background(), p))
	if err := runSession(c, func() snapshot {
		return snapshot{InGit: true, Configured: true}
	}); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 1 || ran[0] != "list" {
		t.Fatalf("ran %v", ran)
	}
}

func TestRunSessionHelpThenQuit(t *testing.T) {
	var ran []string
	orig := dispatchAction
	dispatchAction = func(_ *cobra.Command, action string, _ ...string) error {
		ran = append(ran, action)
		return nil
	}
	defer func() { dispatchAction = orig }()

	var buf bytes.Buffer
	c := NewCommand()
	catalog.AttachObjectHelp(c, "repo")
	c.SetOut(&buf)
	p := &sessionPrompter{picks: []string{"help", "quit"}}
	c.SetContext(prompt.WithPrompter(context.Background(), p))
	if err := runSession(c, func() snapshot {
		return snapshot{InGit: true, Configured: true}
	}); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 0 {
		t.Fatalf("ran %v", ran)
	}
	if len(p.asked) != 2 {
		t.Fatalf("asked=%v", p.asked)
	}
	if !strings.Contains(buf.String(), "repo — manage repositories") {
		t.Fatalf("usage=%q", buf.String())
	}
}

type sessionPrompter struct {
	picks []string
	asked []string
	idx   int
}

func (p *sessionPrompter) String(string, string) (string, error) { return "", nil }
func (p *sessionPrompter) Confirm(string, bool) (bool, error)    { return false, nil }
func (p *sessionPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}
func (p *sessionPrompter) Pick(label string, _ []prompt.Choice, _ string) (string, error) {
	p.asked = append(p.asked, label)
	if p.idx < len(p.picks) {
		v := p.picks[p.idx]
		p.idx++
		return v, nil
	}
	return "quit", nil
}
func (p *sessionPrompter) Required(string, string) error               { return nil }
func (p *sessionPrompter) EditOrAccept(string, string) (string, error) { return "", nil }
func (p *sessionPrompter) Optional(string, string) (string, error)     { return "", nil }
func (p *sessionPrompter) Closed(string, []string, string, bool) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *sessionPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}
