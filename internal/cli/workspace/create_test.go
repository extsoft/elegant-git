package workspace

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/bees-hive/elegant-git/internal/cli/argspec"
	"github.com/bees-hive/elegant-git/internal/prompt"
)

func TestCreateSpecKeepsCLIArgs(t *testing.T) {
	var name, userName, userEmail, signingKey, gpgProgram, editor string
	spec := createSpec(&name, &userName, &userEmail, &signingKey, &gpgProgram, &editor)
	p := &createRecordingPrompter{}
	args := []string{"dz", "D", "d@x.com", "KEY", "/usr/bin/gpg", "vim"}
	if err := argspec.Resolve(context.Background(), p, args, spec); err != nil {
		t.Fatal(err)
	}
	if name != "dz" || userName != "D" || userEmail != "d@x.com" {
		t.Fatalf("required: name=%q userName=%q userEmail=%q", name, userName, userEmail)
	}
	if signingKey != "KEY" || gpgProgram != "/usr/bin/gpg" || editor != "vim" {
		t.Fatalf("optional: signingKey=%q gpgProgram=%q editor=%q", signingKey, gpgProgram, editor)
	}
	if p.stringIdx != 0 || len(p.edits) != 0 {
		t.Fatal("unexpected prompts")
	}
}

func TestCreateSpecMissingRequiredNonInteractive(t *testing.T) {
	var name, userName, userEmail, signingKey, gpgProgram, editor string
	spec := createSpec(&name, &userName, &userEmail, &signingKey, &gpgProgram, &editor)
	err := argspec.Resolve(context.Background(), prompt.NewNonInteractive(), []string{"dz"}, spec)
	if !argspec.IsMissingRequired(err) {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "user-name") {
		t.Fatalf("got %v", err)
	}
}

func TestCreateHelp(t *testing.T) {
	c := newCreateCommand()
	var buf bytes.Buffer
	c.SetOut(&buf)
	c.SetArgs([]string{"--help"})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"Usage:",
		"create <name> <user-name> <user-email>",
		"Creates a Git workspace",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	for _, ban := range []string{"--name", "--user-name", "--user-email"} {
		if strings.Contains(out, ban) {
			t.Fatalf("unexpected %q in %q", ban, out)
		}
	}
}

type createRecordingPrompter struct {
	strings   []string
	stringIdx int
	edits     []string
}

func (r *createRecordingPrompter) String(question, defaultVal string) (string, error) {
	if r.stringIdx < len(r.strings) {
		v := r.strings[r.stringIdx]
		r.stringIdx++
		return v, nil
	}
	return defaultVal, nil
}

func (r *createRecordingPrompter) Confirm(string, bool) (bool, error) { return false, nil }

func (r *createRecordingPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}

func (r *createRecordingPrompter) Pick(string, []prompt.Choice, string) (string, error) {
	return "", prompt.ErrNonInteractive
}

func (r *createRecordingPrompter) Required(string, string) error { return nil }

func (r *createRecordingPrompter) EditOrAccept(label, suggested string) (string, error) {
	r.edits = append(r.edits, label)
	return suggested, nil
}

func (r *createRecordingPrompter) Optional(string, string) (string, error) { return "", nil }

func (r *createRecordingPrompter) Closed(string, []string, string, bool) (string, error) {
	return "", prompt.ErrNonInteractive
}

func (r *createRecordingPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}
