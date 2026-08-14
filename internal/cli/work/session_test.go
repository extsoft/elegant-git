package work

import (
	"bytes"
	"context"
	"os"
	"slices"
	"strings"
	"testing"

	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/bees-hive/elegant-git/internal/text"
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

func TestRunSessionDirtyProtectedStart(t *testing.T) {
	var ran []string
	orig := dispatchAction
	dispatchAction = func(_ *cobra.Command, action string) error {
		ran = append(ran, action)
		return nil
	}
	defer func() { dispatchAction = orig }()

	var buf bytes.Buffer
	text.SetOutput(&buf)
	defer text.SetOutput(os.Stdout)

	c := NewCommand()
	c.SetContext(prompt.WithPrompter(context.Background(), &sessionPrompter{}))
	err := runSession(c, func() snapshot {
		return snapshot{Branch: "main", Protected: true, Dirty: true}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ran, []string{"start"}) {
		t.Fatalf("ran %v", ran)
	}
	out := buf.String()
	if !strings.Contains(out, "==>> Detection action...") || !strings.Contains(out, "selected: git elegant work start") {
		t.Fatalf("eval=%q", out)
	}
}

func TestRunSessionListThenAskQuit(t *testing.T) {
	var ran []string
	orig := dispatchAction
	dispatchAction = func(_ *cobra.Command, action string) error {
		ran = append(ran, action)
		return nil
	}
	defer func() { dispatchAction = orig }()

	var buf bytes.Buffer
	text.SetOutput(&buf)
	defer text.SetOutput(os.Stdout)

	p := &sessionPrompter{closed: []string{"quit"}}
	c := NewCommand()
	c.SetContext(prompt.WithPrompter(context.Background(), p))
	snap := snapshot{Branch: "main", Protected: true, HasUpstream: true}
	err := runSession(c, func() snapshot { return snap })
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ran, []string{"list"}) {
		t.Fatalf("ran %v", ran)
	}
	if len(p.asked) != 1 || p.asked[0] != "What next?" {
		t.Fatalf("asked=%v", p.asked)
	}
	out := buf.String()
	if !strings.Contains(out, "==>> Detection action...") || !strings.Contains(out, "selected: git elegant work list") || !strings.Contains(out, "selected: ask") {
		t.Fatalf("eval=%q", out)
	}
}

func TestRunSessionAskPush(t *testing.T) {
	var ran []string
	orig := dispatchAction
	dispatchAction = func(_ *cobra.Command, action string) error {
		ran = append(ran, action)
		return nil
	}
	defer func() { dispatchAction = orig }()

	p := &sessionPrompter{closed: []string{"push"}}
	c := NewCommand()
	c.SetContext(prompt.WithPrompter(context.Background(), p))
	err := runSession(c, func() snapshot {
		return snapshot{Branch: "feat", UniqueCommits: true, HasUpstream: true, Ahead: 2}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ran, []string{"push"}) {
		t.Fatalf("ran %v", ran)
	}
}

type sessionPrompter struct {
	closed []string
	idx    int
	asked  []string
}

func (p *sessionPrompter) String(string, string) (string, error) { return "", nil }
func (p *sessionPrompter) Confirm(string, bool) (bool, error)    { return false, nil }
func (p *sessionPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}
func (p *sessionPrompter) Pick(string, []prompt.Choice) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *sessionPrompter) Required(string, string) error { return nil }
func (p *sessionPrompter) EditOrAccept(string, string) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *sessionPrompter) Optional(string, string) (string, error) { return "", nil }
func (p *sessionPrompter) Closed(question string, _ []string, def string, _ bool) (string, error) {
	p.asked = append(p.asked, question)
	if p.idx < len(p.closed) {
		v := p.closed[p.idx]
		p.idx++
		return v, nil
	}
	return def, nil
}
func (p *sessionPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}
