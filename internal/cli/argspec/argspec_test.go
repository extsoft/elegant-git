package argspec

import (
	"context"
	"errors"
	"testing"

	"github.com/bees-hive/elegant-git/internal/prompt"
)

type recordingPrompter struct {
	strings      []string
	editValues   []string
	optValues    []string
	pickValues   []string
	closedValues []string
	edits        []editCall
	optionals    []editCall
	closed       []string
	stringIdx    int
	editIdx      int
	optIdx       int
	pickIdx      int
	closedIdx    int
}

type editCall struct {
	label, suggested string
}

func (r *recordingPrompter) String(question, defaultVal string) (string, error) {
	if r.stringIdx < len(r.strings) {
		v := r.strings[r.stringIdx]
		r.stringIdx++
		return v, nil
	}
	return defaultVal, nil
}

func (r *recordingPrompter) Confirm(string, bool) (bool, error) { return false, nil }

func (r *recordingPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}

func (r *recordingPrompter) Pick(string, []prompt.Choice, string) (string, error) {
	if r.pickIdx < len(r.pickValues) {
		v := r.pickValues[r.pickIdx]
		r.pickIdx++
		return v, nil
	}
	return "", errors.New("no pick value")
}

func (r *recordingPrompter) Required(string, string) error { return nil }

func (r *recordingPrompter) EditOrAccept(label, suggested string) (string, error) {
	r.edits = append(r.edits, editCall{label, suggested})
	if r.editIdx < len(r.editValues) {
		v := r.editValues[r.editIdx]
		r.editIdx++
		if v != "" {
			return v, nil
		}
	}
	return suggested, nil
}

func (r *recordingPrompter) Optional(label, suggested string) (string, error) {
	r.optionals = append(r.optionals, editCall{label, suggested})
	if r.optIdx < len(r.optValues) {
		v := r.optValues[r.optIdx]
		r.optIdx++
		return v, nil
	}
	return "", nil
}

func (r *recordingPrompter) Closed(question string, _ []string, def string, required bool) (string, error) {
	r.closed = append(r.closed, question)
	if r.closedIdx < len(r.closedValues) {
		v := r.closedValues[r.closedIdx]
		r.closedIdx++
		return v, nil
	}
	if def != "" {
		return def, nil
	}
	if !required {
		return "", nil
	}
	return "", errors.New("no closed value")
}

func (r *recordingPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}

func TestResolveAllRequiredPresent(t *testing.T) {
	name := "feature"
	from := "main"
	spec := Spec{Inputs: []Input{
		PositionalInput("name", 0, true, "Branch name", &name, nil),
		PositionalInput("from-ref", 1, false, "Start from ref", &from, nil),
	}}
	p := &recordingPrompter{}
	if err := Resolve(context.Background(), p, []string{"feature", "main"}, spec); err != nil {
		t.Fatal(err)
	}
	if len(p.edits) != 0 {
		t.Fatalf("expected no optional prompts when all args on CLI, got %+v", p.edits)
	}
}

func TestResolveSkipOptionalWhenRequiredOnCLI(t *testing.T) {
	var name, from string
	complete := func(context.Context) ([]Choice, error) {
		return []Choice{{Value: "main"}, {Value: "develop"}}, nil
	}
	spec := Spec{Inputs: []Input{
		PositionalInput("name", 0, true, "Branch name", &name, nil),
		PositionalInputWithComplete("from-ref", 1, false, "Start from ref", &from, func() string { return "main" }, complete, true),
	}}
	p := &recordingPrompter{pickValues: []string{"should-not-ask"}}
	if err := Resolve(context.Background(), p, []string{"feature"}, spec); err != nil {
		t.Fatal(err)
	}
	if name != "feature" || from != "" {
		t.Fatalf("name=%q from=%q", name, from)
	}
	if p.pickIdx != 0 {
		t.Fatalf("unexpected pick prompts: pickIdx=%d", p.pickIdx)
	}
}

func TestResolveMissingRequiredNonInteractive(t *testing.T) {
	var name string
	spec := Spec{Inputs: []Input{
		PositionalInput("name", 0, true, "Branch name", &name, nil),
	}}
	err := Resolve(context.Background(), prompt.NewNonInteractive(), nil, spec)
	if !IsMissingRequired(err) {
		t.Fatalf("got %v", err)
	}
}

func TestResolveMissingRequiredInteractive(t *testing.T) {
	var name, from string
	spec := Spec{Inputs: []Input{
		PositionalInput("name", 0, true, "Branch name", &name, nil),
		PositionalInput("from-ref", 1, false, "Start from ref", &from, func() string { return "main" }),
	}}
	p := &recordingPrompter{strings: []string{"feature"}, optValues: []string{"develop"}}
	if err := Resolve(context.Background(), p, nil, spec); err != nil {
		t.Fatal(err)
	}
	if name != "feature" || from != "develop" {
		t.Fatalf("name=%q from=%q", name, from)
	}
}

func TestResolveCompletePickRequired(t *testing.T) {
	var branch string
	complete := func(context.Context) ([]Choice, error) {
		return []Choice{{Value: "main"}, {Value: "feature"}}, nil
	}
	spec := Spec{Inputs: []Input{
		PositionalInputWithComplete("branch", 0, true, "Branch to accept", &branch, nil, complete, true),
	}}
	p := &recordingPrompter{pickValues: []string{"feature"}}
	if err := Resolve(context.Background(), p, nil, spec); err != nil {
		t.Fatal(err)
	}
	if branch != "feature" {
		t.Fatalf("branch=%q", branch)
	}
}

func TestResolveCompletePickTabbedValue(t *testing.T) {
	var branch string
	complete := func(context.Context) ([]Choice, error) {
		return []Choice{{Value: "origin/main", Description: "abc123"}}, nil
	}
	spec := Spec{Inputs: []Input{
		PositionalInputWithComplete("branch", 0, true, "Branch", &branch, nil, complete, true),
	}}
	p := &recordingPrompter{pickValues: []string{"origin/main\tabc123"}}
	if err := Resolve(context.Background(), p, nil, spec); err != nil {
		t.Fatal(err)
	}
	if branch != "origin/main" {
		t.Fatalf("branch=%q", branch)
	}
}

func TestResolveOmitInteractiveOptional(t *testing.T) {
	var name string
	in := PositionalInput("name", 0, false, "Workspace name", &name, nil)
	in.OmitInteractive = true
	spec := Spec{Inputs: []Input{in}}
	p := &recordingPrompter{strings: []string{"should-not-ask"}}
	if err := Resolve(context.Background(), p, nil, spec); err != nil {
		t.Fatal(err)
	}
	if name != "" {
		t.Fatalf("name=%q", name)
	}
	if p.stringIdx != 0 {
		t.Fatalf("unexpected prompts: stringIdx=%d", p.stringIdx)
	}
}

func TestResolveOmitInteractiveKeepsCLIArg(t *testing.T) {
	var name string
	complete := func(context.Context) ([]Choice, error) {
		return []Choice{{Value: "dz"}, {Value: "work"}}, nil
	}
	in := PositionalInputWithComplete("name", 0, false, "Workspace name", &name, nil, complete, true)
	in.OmitInteractive = true
	spec := Spec{Inputs: []Input{in}}
	p := &recordingPrompter{}
	if err := Resolve(context.Background(), p, []string{"dz"}, spec); err != nil {
		t.Fatal(err)
	}
	if name != "dz" {
		t.Fatalf("name=%q", name)
	}
	if p.pickIdx != 0 {
		t.Fatalf("unexpected pick prompts: pickIdx=%d", p.pickIdx)
	}
}

func TestResolveTwoChoicesUsesPick(t *testing.T) {
	var hookType string
	complete := func(context.Context) ([]Choice, error) {
		return []Choice{{Value: "ahead"}, {Value: "after"}}, nil
	}
	spec := Spec{Inputs: []Input{
		PositionalInputWithComplete("hook-type", 0, true, "Hook type", &hookType, nil, complete, true),
	}}
	p := &recordingPrompter{pickValues: []string{"after"}, closedValues: []string{"should-not-closed"}}
	if err := Resolve(context.Background(), p, nil, spec); err != nil {
		t.Fatal(err)
	}
	if hookType != "after" {
		t.Fatalf("hookType=%q", hookType)
	}
	if p.pickIdx != 1 {
		t.Fatalf("expected pick, got pickIdx=%d", p.pickIdx)
	}
	if len(p.closed) != 0 {
		t.Fatalf("closed calls=%v", p.closed)
	}
}

func TestResolveOptionalTextLeavesUnset(t *testing.T) {
	var name, key string
	spec := Spec{Inputs: []Input{
		PositionalInput("name", 0, true, "Workspace name", &name, nil),
		PositionalInput("signing-key", 1, false, "Signing key", &key, func() string { return "ABC123" }),
	}}
	p := &recordingPrompter{strings: []string{"dz"}, optValues: []string{""}}
	if err := Resolve(context.Background(), p, nil, spec); err != nil {
		t.Fatal(err)
	}
	if name != "dz" || key != "" {
		t.Fatalf("name=%q key=%q", name, key)
	}
	if len(p.optionals) != 1 || p.optionals[0].suggested != "ABC123" {
		t.Fatalf("optionals=%+v", p.optionals)
	}
	if len(p.edits) != 0 {
		t.Fatalf("unexpected EditOrAccept: %+v", p.edits)
	}
}

func TestResolveRequiredSuggestUsesEditOrAccept(t *testing.T) {
	var name string
	spec := Spec{Inputs: []Input{
		PositionalInput("name", 0, true, "Workspace name", &name, func() string { return "alice" }),
	}}
	p := &recordingPrompter{editValues: []string{""}}
	if err := Resolve(context.Background(), p, nil, spec); err != nil {
		t.Fatal(err)
	}
	if name != "alice" {
		t.Fatalf("name=%q", name)
	}
	if len(p.edits) != 1 || p.edits[0].suggested != "alice" {
		t.Fatalf("edits=%+v", p.edits)
	}
}

func TestIsMissingRequired(t *testing.T) {
	if !IsMissingRequired(&ErrMissingRequired{Names: []string{"a"}}) {
		t.Fatal("expected true")
	}
	if IsMissingRequired(errors.New("other")) {
		t.Fatal("expected false")
	}
}
