package doctor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/prompt"
)

func TestRunNonInteractivePrintsOnly(t *testing.T) {
	applied := false
	findings := []Finding{{
		Problem: "broken",
		Repair:  "fix it",
		Apply: func() error {
			applied = true
			return nil
		},
	}}
	var buf bytes.Buffer
	changed, err := Run(&buf, prompt.NewNonInteractive(), findings)
	if err != nil {
		t.Fatal(err)
	}
	if changed || applied {
		t.Fatalf("non-interactive must not apply: changed=%v applied=%v", changed, applied)
	}
	if !strings.Contains(buf.String(), "problem: broken") || !strings.Contains(buf.String(), "repair: fix it") {
		t.Fatalf("out=%s", buf.String())
	}
}

func TestRunSkipsAdvisoryWithoutAsking(t *testing.T) {
	p := &runPrompter{confirm: true}
	findings := []Finding{{
		Problem: "hint only",
		Repair:  "run something else",
	}}
	var buf bytes.Buffer
	changed, err := Run(&buf, p, findings)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("advisory must not change state")
	}
	if p.confirmCalls != 0 {
		t.Fatalf("confirmCalls=%d", p.confirmCalls)
	}
	if !strings.Contains(buf.String(), "hint only") {
		t.Fatalf("out=%s", buf.String())
	}
}

func TestRunAppliesOnConfirm(t *testing.T) {
	applied := false
	p := &runPrompter{confirm: true}
	findings := []Finding{{
		Problem: "broken",
		Repair:  "fix it",
		Apply: func() error {
			applied = true
			return nil
		},
	}}
	changed, err := Run(&bytes.Buffer{}, p, findings)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || !applied {
		t.Fatalf("changed=%v applied=%v", changed, applied)
	}
}

func TestRunDeclineLeavesUnchanged(t *testing.T) {
	applied := false
	p := &runPrompter{confirm: false}
	findings := []Finding{{
		Problem: "broken",
		Repair:  "fix it",
		Apply: func() error {
			applied = true
			return nil
		},
	}}
	changed, err := Run(&bytes.Buffer{}, p, findings)
	if err != nil {
		t.Fatal(err)
	}
	if changed || applied {
		t.Fatalf("changed=%v applied=%v", changed, applied)
	}
}

func TestRunDecideSkipsConfirm(t *testing.T) {
	applied := false
	p := &runPrompter{confirm: false}
	findings := []Finding{{
		Problem: "broken",
		Repair:  "choose",
		Decide:  true,
		Apply: func() error {
			applied = true
			return nil
		},
	}}
	changed, err := Run(&bytes.Buffer{}, p, findings)
	if err != nil {
		t.Fatal(err)
	}
	if p.confirmCalls != 0 {
		t.Fatalf("confirmCalls=%d", p.confirmCalls)
	}
	if !changed || !applied {
		t.Fatalf("changed=%v applied=%v", changed, applied)
	}
}

func TestRunDecideSkippedLeavesUnchanged(t *testing.T) {
	p := &runPrompter{}
	findings := []Finding{{
		Problem: "broken",
		Repair:  "choose",
		Decide:  true,
		Apply:   func() error { return ErrSkipped },
	}}
	changed, err := Run(&bytes.Buffer{}, p, findings)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("skip must not mark changed")
	}
}

type runPrompter struct {
	confirm      bool
	confirmCalls int
}

func (p *runPrompter) String(string, string) (string, error) { return "", nil }
func (p *runPrompter) Confirm(string, bool) (bool, error) {
	p.confirmCalls++
	return p.confirm, nil
}
func (p *runPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}
func (p *runPrompter) Pick(string, []prompt.Choice, string) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *runPrompter) Required(string, string) error               { return nil }
func (p *runPrompter) EditOrAccept(string, string) (string, error) { return "", nil }
func (p *runPrompter) Optional(string, string) (string, error)     { return "", nil }
func (p *runPrompter) Closed(string, []string, string, bool) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *runPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}
