package workspace

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

func TestRunBareNonInteractive(t *testing.T) {
	c := NewCommand()
	c.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
	err := runBare(c, nil)
	if !cliruntime.IsUsageError(err) {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "action is required") {
		t.Fatalf("msg=%v", err)
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
		return snapshot{InGit: true, Linked: true, WorkspaceName: "github", WorkspaceCount: 1}
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
	if !strings.Contains(out, "workspace linked? yes (github)") {
		t.Fatalf("eval=%q", out)
	}
}

func TestRunSessionDispatchStatus(t *testing.T) {
	var ran []string
	orig := dispatchAction
	dispatchAction = func(_ *cobra.Command, action string, _ ...string) error {
		ran = append(ran, action)
		return nil
	}
	defer func() { dispatchAction = orig }()

	p := &sessionPrompter{picks: []string{"status"}}
	c := NewCommand()
	c.SetContext(prompt.WithPrompter(context.Background(), p))
	if err := runSession(c, func() snapshot {
		return snapshot{InGit: true, Linked: true, WorkspaceName: "github", WorkspaceCount: 1}
	}); err != nil {
		t.Fatal(err)
	}
	if len(ran) != 1 || ran[0] != "status" {
		t.Fatalf("ran %v", ran)
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
