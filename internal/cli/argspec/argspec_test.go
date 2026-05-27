package argspec

import (
	"errors"
	"testing"

	"github.com/bees-hive/elegant-git/internal/prompt"
)

type recordingPrompter struct {
	strings    []string
	editValues []string
	edits      []editCall
	stringIdx  int
	editIdx    int
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

func (r *recordingPrompter) Confirm(string) (bool, error) { return false, nil }

func (r *recordingPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
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

func (r *recordingPrompter) BatchChoice(string) (prompt.BatchDecision, error) {
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
	if err := Resolve(p, []string{"feature", "main"}, spec); err != nil {
		t.Fatal(err)
	}
	if len(p.edits) != 0 {
		t.Fatalf("unexpected prompts: %+v", p.edits)
	}
}

func TestResolveMissingRequiredNonInteractive(t *testing.T) {
	var name string
	spec := Spec{Inputs: []Input{
		PositionalInput("name", 0, true, "Branch name", &name, nil),
	}}
	err := Resolve(prompt.NewNonInteractive(), nil, spec)
	if !IsMissingRequired(err) {
		t.Fatalf("got %v", err)
	}
	var mr *ErrMissingRequired
	if !errors.As(err, &mr) {
		t.Fatalf("got %T", err)
	}
	if len(mr.Names) != 1 || mr.Names[0] != "name" {
		t.Fatalf("names: %+v", mr.Names)
	}
}

func TestResolveMissingRequiredInteractive(t *testing.T) {
	var name, from string
	spec := Spec{Inputs: []Input{
		PositionalInput("name", 0, true, "Branch name", &name, nil),
		PositionalInput("from-ref", 1, false, "Start from ref", &from, func() string { return "main" }),
	}}
	p := &recordingPrompter{strings: []string{"feature"}, editValues: []string{"develop"}}
	if err := Resolve(p, nil, spec); err != nil {
		t.Fatal(err)
	}
	if name != "feature" {
		t.Fatalf("name=%q", name)
	}
	if from != "develop" {
		t.Fatalf("from=%q", from)
	}
	if len(p.edits) != 1 {
		t.Fatalf("edits: %+v", p.edits)
	}
}

func TestResolveHydratesPositionals(t *testing.T) {
	var name string
	spec := Spec{Inputs: []Input{
		PositionalInput("name", 0, true, "Branch name", &name, nil),
	}}
	if err := Resolve(&recordingPrompter{}, []string{"x"}, spec); err != nil {
		t.Fatal(err)
	}
	if name != "x" {
		t.Fatalf("got %q", name)
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
