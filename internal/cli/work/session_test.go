package work

import (
	"bytes"
	"context"
	"os"
	"slices"
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

func TestRunSessionDirtyProtectedStart(t *testing.T) {
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
	if !strings.Contains(out, "==>> Detection action...") || !strings.Contains(out, "selected: eg work start") {
		t.Fatalf("eval=%q", out)
	}
}

func TestRunSessionListThenAskQuit(t *testing.T) {
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
	snap := snapshot{Branch: "main", Protected: true, HasUpstream: true}
	err := runSession(c, func() snapshot { return snap })
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ran, []string{"list"}) {
		t.Fatalf("ran %v", ran)
	}
	if len(p.asked) != 1 || p.asked[0] != "What now" {
		t.Fatalf("asked=%v", p.asked)
	}
	if len(p.pickChoices) != 1 || p.pickChoices[0][0].Description == "" {
		t.Fatalf("expected action descriptions, got %+v", p.pickChoices)
	}
	out := buf.String()
	if !strings.Contains(out, "==>> Detection action...") || !strings.Contains(out, "selected: eg work list") || !strings.Contains(out, "selected: ask") {
		t.Fatalf("eval=%q", out)
	}
}

func TestRunSessionAskPush(t *testing.T) {
	var ran []string
	orig := dispatchAction
	dispatchAction = func(_ *cobra.Command, action string, _ ...string) error {
		ran = append(ran, action)
		return nil
	}
	defer func() { dispatchAction = orig }()

	p := &sessionPrompter{picks: []string{"push"}}
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

func TestRunSessionAskAcceptUsesCurrentBranch(t *testing.T) {
	var ran []string
	var args [][]string
	orig := dispatchAction
	dispatchAction = func(_ *cobra.Command, action string, a ...string) error {
		ran = append(ran, action)
		args = append(args, append([]string(nil), a...))
		return nil
	}
	defer func() { dispatchAction = orig }()

	p := &sessionPrompter{picks: []string{"accept"}}
	c := NewCommand()
	c.SetContext(prompt.WithPrompter(context.Background(), p))
	err := runSession(c, func() snapshot {
		return snapshot{Branch: "feat", UniqueCommits: true, HasUpstream: true, Ahead: 2}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ran, []string{"accept"}) {
		t.Fatalf("ran %v", ran)
	}
	if len(args) != 1 || !slices.Equal(args[0], []string{"feat"}) {
		t.Fatalf("args %v", args)
	}
}

func TestRunSessionAskAcceptProtectedDoesNotPassBranch(t *testing.T) {
	var args [][]string
	orig := dispatchAction
	dispatchAction = func(_ *cobra.Command, _ string, a ...string) error {
		args = append(args, append([]string(nil), a...))
		return nil
	}
	defer func() { dispatchAction = orig }()

	p := &sessionPrompter{picks: []string{"accept"}}
	c := NewCommand()
	c.SetContext(prompt.WithPrompter(context.Background(), p))
	err := runSession(c, func() snapshot {
		return snapshot{Branch: "main", Protected: true, UniqueCommits: true}
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 1 || len(args[0]) != 0 {
		t.Fatalf("args %v", args)
	}
}

type sessionPrompter struct {
	picks       []string
	idx         int
	asked       []string
	pickChoices [][]prompt.Choice
}

func (p *sessionPrompter) String(string, string) (string, error) { return "", nil }
func (p *sessionPrompter) Confirm(string, bool) (bool, error)    { return false, nil }
func (p *sessionPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}
func (p *sessionPrompter) Pick(question string, choices []prompt.Choice, def string) (string, error) {
	p.asked = append(p.asked, question)
	p.pickChoices = append(p.pickChoices, append([]prompt.Choice(nil), choices...))
	if p.idx < len(p.picks) {
		v := p.picks[p.idx]
		p.idx++
		return v, nil
	}
	if def != "" {
		return def, nil
	}
	return "", prompt.ErrNonInteractive
}
func (p *sessionPrompter) Required(string, string) error { return nil }
func (p *sessionPrompter) EditOrAccept(string, string) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *sessionPrompter) Optional(string, string) (string, error) { return "", nil }
func (p *sessionPrompter) Closed(string, []string, string, bool) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *sessionPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}
