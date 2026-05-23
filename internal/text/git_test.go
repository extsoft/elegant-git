package text

import (
	"bytes"
	"strings"
	"testing"
)

func TestSuggestGitAddCommit(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	SuggestGitAddCommit(
		[]string{".config/elegant-git/hooks/work-start-ahead"},
		[]string{".workflows/start-work-ahead"},
		"Migrate Elegant Git hooks",
	)
	got := buf.String()
	if !strings.Contains(got, "git add .config/elegant-git/hooks/work-start-ahead") {
		t.Fatalf("missing new path: %q", got)
	}
	if !strings.Contains(got, "git add -u .workflows/start-work-ahead") {
		t.Fatalf("missing old path: %q", got)
	}
	if !strings.Contains(got, `git commit -m "Migrate Elegant Git hooks"`) {
		t.Fatalf("missing commit message: %q", got)
	}
}
