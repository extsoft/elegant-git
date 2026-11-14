package prompt

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/extsoft/elegant-git/internal/text"
)

func TestBuildValidatedScreen(t *testing.T) {
	lines, col := buildValidatedScreen("Branch name", "main", "branch 'main' already exists")
	if len(lines) != 2 || lines[0] != "Branch name: main" || lines[1] != "branch 'main' already exists" {
		t.Fatalf("lines=%q", lines)
	}
	if col != len("Branch name: main")+1 {
		t.Fatalf("cursorCol=%d", col)
	}

	lines, col = buildValidatedScreen("Branch name", "mainx", "")
	if len(lines) != 1 || lines[0] != "Branch name: mainx" {
		t.Fatalf("clear lines=%q", lines)
	}
	if col != len("Branch name: mainx")+1 {
		t.Fatalf("cursorCol=%d", col)
	}

	value := "café"
	lines, col = buildValidatedScreen("Branch name", value, "")
	wantCol := displayWidth("Branch name: "+value) + 1
	if lines[0] != "Branch name: "+value || col != wantCol {
		t.Fatalf("line=%q col=%d wantCol=%d", lines[0], col, wantCol)
	}
}

func TestDisplayWidth(t *testing.T) {
	if displayWidth("a") != 1 || displayWidth("é") != 1 || displayWidth("e\u0301") != 1 {
		t.Fatalf("narrow a=%d e=%d combining=%d", displayWidth("a"), displayWidth("é"), displayWidth("e\u0301"))
	}
	if displayWidth("中") != 2 || displayWidth("😀") != 2 {
		t.Fatalf("wide=%d emoji=%d", displayWidth("中"), displayWidth("😀"))
	}
	_, col := buildValidatedScreen("N", "中", "")
	if col != displayWidth("N: 中")+1 {
		t.Fatalf("col=%d", col)
	}
}

func TestWriteValidatedScreenPaintsProblem(t *testing.T) {
	out := &bytes.Buffer{}
	lines, col := buildValidatedScreen("Branch name", "main", "branch 'main' already exists")
	writeValidatedScreen(out, lines, 0, "Branch name", "main", "branch 'main' already exists", col)
	got := out.String()
	if !strings.Contains(got, "\033[1;34mBranch name: \033[m"+"main") {
		t.Fatalf("prompt=%q", got)
	}
	if !strings.Contains(got, "\033[1;31mbranch 'main' already exists\033[m") {
		t.Fatalf("status=%q", got)
	}
	if !strings.Contains(got, "\r\n") || !strings.Contains(got, "\033[J") {
		t.Fatalf("redraw=%q", got)
	}
	if !strings.Contains(got, "\033[2A") {
		t.Fatalf("cursor up=%q", got)
	}

	out.Reset()
	lines, col = buildValidatedScreen("Branch name", "mainx", "")
	writeValidatedScreen(out, lines, 2, "Branch name", "mainx", "", col)
	got = out.String()
	if strings.Contains(got, "already exists") {
		t.Fatalf("cleared frame still has the problem: %q", got)
	}
	if !strings.Contains(got, "\033[2A") {
		t.Fatalf("single-line frame=%q", got)
	}
}

func TestValidatedLineScriptedProblemEnterAndClear(t *testing.T) {
	got, screen, err := runValidatedLoop(t, "main\rx\r", takenName)
	if err != nil {
		t.Fatal(err)
	}
	if got != "mainx" {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(screen, "branch 'main' already exists") {
		t.Fatalf("screen=%q", screen)
	}
	cleared := false
	for _, frame := range strings.Split(screen, "\033[J") {
		if strings.Contains(frame, "mainx") && !strings.Contains(frame, "already exists") {
			cleared = true
		}
	}
	if !cleared {
		t.Fatalf("no clear frame: %q", screen)
	}
}

func TestValidatedLineEnterIgnoredUntilUsable(t *testing.T) {
	got, _, err := runValidatedLoop(t, "\ra\r", takenName)
	if err != nil {
		t.Fatal(err)
	}
	if got != "a" {
		t.Fatalf("got %q", got)
	}
}

func TestValidatedLineBackspaceAndUTF8(t *testing.T) {
	got, screen, err := runValidatedLoop(t, "ab\x7f\r", func(s string) string {
		if s == "ab" {
			return "no"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "a" {
		t.Fatalf("backspace got %q", got)
	}
	if !strings.Contains(screen, "no") {
		t.Fatalf("screen=%q", screen)
	}

	got, _, err = runValidatedLoop(t, "caf\xc3\xa9\r", takenName)
	if err != nil {
		t.Fatal(err)
	}
	if got != "café" {
		t.Fatalf("utf8 got %q", got)
	}
	got, _, err = runValidatedLoop(t, "caf\xc3\xa9\x7f\r", takenName)
	if err != nil {
		t.Fatal(err)
	}
	if got != "caf" {
		t.Fatalf("utf8 backspace got %q", got)
	}
}

func TestValidatedLineCancel(t *testing.T) {
	for _, in := range []string{"\x03", "\x04", "\x1b"} {
		_, _, err := runValidatedLoop(t, in, takenName)
		if !errors.Is(err, ErrUserCancelled) {
			t.Fatalf("input %q err=%v", in, err)
		}
	}
}

func TestValidatedLineArrowsDoNotCancel(t *testing.T) {
	got, _, err := runValidatedLoop(t, "\x1b[D\x1b[Aa\r", takenName)
	if err != nil {
		t.Fatal(err)
	}
	if got != "a" {
		t.Fatalf("got %q", got)
	}
}

func TestValidatedLineDropsCSIAndSS3(t *testing.T) {
	got, _, err := runValidatedLoop(t, "\x1b[3~\x1b[5~\x1b[6~\x1b[1;5C\x1bOPab\r", takenName)
	if err != nil {
		t.Fatal(err)
	}
	if got != "ab" {
		t.Fatalf("got %q", got)
	}
}

func TestDropLastWord(t *testing.T) {
	tests := []struct{ in, want string }{
		{"feature/ops-2114", "feature/ops-"},
		{"feature/ops-", "feature/"},
		{"feature/", ""},
		{"feature_ops", "feature_"},
		{"feature.ops", "feature."},
		{"a--b", "a--"},
		{"ab cd", "ab "},
		{"hello", ""},
		{"  x", "  "},
		{"foo/-", ""},
		{"ops-2114.", "ops-"},
		{"café-x", "café-"},
	}
	for _, tc := range tests {
		if got := dropLastWord(tc.in); got != tc.want {
			t.Errorf("dropLastWord(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidatedLineAltBackspaceStopsAtSeparator(t *testing.T) {
	got, _, err := runValidatedLoop(t, "feature/ops-2114\x1b\x7f\r", func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if got != "feature/ops-" {
		t.Fatalf("got %q", got)
	}
}

func TestValidatedLineEscDelDeletesWord(t *testing.T) {
	got, _, err := runValidatedLoop(t, "ab cd\x1b\x7f\r", func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if got != "ab " {
		t.Fatalf("got %q", got)
	}
	got, _, err = runValidatedLoop(t, "hello\x1b\x7f\r", func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestValidatedLineEmptyUntouchedHidesRequired(t *testing.T) {
	got, screen, err := runValidatedLoop(t, "\ra\r", takenName)
	if err != nil {
		t.Fatal(err)
	}
	if got != "a" {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(screen, "required") {
		t.Fatalf("untouched empty showed a problem: %q", screen)
	}
	_, screen, err = runValidatedLoop(t, "x\x7f\rb\r", takenName)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(screen, "branch name is required") {
		t.Fatalf("cleared line hid the problem: %q", screen)
	}
}

func TestValidatedLinePrefill(t *testing.T) {
	got, screen, err := runValidatedLoopFrom(t, "main", "\rx\r", takenName)
	if err != nil {
		t.Fatal(err)
	}
	if got != "mainx" {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(screen, "branch 'main' already exists") || !strings.Contains(screen, "main") {
		t.Fatalf("screen=%q", screen)
	}
}

func TestValidatedLineInvalidUTF8ThenEnter(t *testing.T) {
	got, _, err := runValidatedLoop(t, "\xc3\ra\r", takenName)
	if err != nil {
		t.Fatal(err)
	}
	if got != "a" {
		t.Fatalf("got %q", got)
	}
	_, _, err = runValidatedLoop(t, "\xc3", takenName)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("err=%v", err)
	}
}

func TestLoneEscCancelsWithoutSwallowingNextByte(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	go func() {
		_, _ = w.Write([]byte{27})
	}()
	type result struct {
		key pickKey
		err error
	}
	done := make(chan result, 1)
	go func() {
		key, err := readValidatedKey(r)
		done <- result{key, err}
	}()
	select {
	case res := <-done:
		if res.err != nil {
			t.Fatal(res.err)
		}
		if res.key.kind != pickKeyCancel {
			t.Fatalf("kind=%v", res.key.kind)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("lone esc did not cancel")
	}
	if _, err := w.Write([]byte{'x'}); err != nil {
		t.Fatal(err)
	}
	key, err := readValidatedKey(r)
	if err != nil {
		t.Fatal(err)
	}
	if key.kind != pickKeyRune || key.rune != 'x' {
		t.Fatalf("key=%+v", key)
	}
}

func TestReadPickKeyLeftArrowCancels(t *testing.T) {
	key := readScriptedKey(t, []byte{27, '[', 'D'}, false)
	if key.kind != pickKeyCancel {
		t.Fatalf("kind=%v", key.kind)
	}
	key = readScriptedKey(t, []byte{27, 'O', 'P'}, false)
	if key.kind != pickKeyNone {
		t.Fatalf("ss3 kind=%v", key.kind)
	}
}

func TestReadValidatedKeyUTF8(t *testing.T) {
	key := readScriptedKey(t, []byte{0xc3, 0xa9}, true)
	if key.kind != pickKeyRune || key.rune != 'é' {
		t.Fatalf("key=%+v", key)
	}
}

func TestValidatedLinePipeUsesSubmitLoop(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	go func() {
		_, _ = io.WriteString(w, "main\nfeature\n")
		_ = w.Close()
	}()
	var shown bytes.Buffer
	text.SetOutput(&shown)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	var out bytes.Buffer
	got, err := NewTTY(r, &out).ValidatedLine("Branch name", "", takenName)
	if err != nil {
		t.Fatal(err)
	}
	if got != "feature" {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(shown.String(), "branch 'main' already exists") {
		t.Fatalf("shown=%q", shown.String())
	}
	if strings.Contains(out.String(), "\033[J") {
		t.Fatalf("raw redraw on a pipe: %q", out.String())
	}
}

func TestValidatedLineFallbackPrompter(t *testing.T) {
	var shown bytes.Buffer
	text.SetOutput(&shown)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	p := &validatedStub{answers: []string{"main", "feature"}}
	got, err := ValidatedLine(p, "Branch name", "", takenName)
	if err != nil {
		t.Fatal(err)
	}
	if got != "feature" || len(p.labels) != 2 || p.labels[0] != "Branch name" {
		t.Fatalf("got %q labels=%v", got, p.labels)
	}
	if !strings.Contains(shown.String(), "branch 'main' already exists") {
		t.Fatalf("shown=%q", shown.String())
	}
}

func TestValidatedLineFallbackRejectedInitialStopsAtEOF(t *testing.T) {
	var shown bytes.Buffer
	text.SetOutput(&shown)
	t.Cleanup(func() { text.SetOutput(os.Stdout) })
	p := NewTTY(strings.NewReader(""), &shown)
	_, err := ValidatedLine(p, "Branch name", "main", takenName)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("err=%v shown=%q", err, shown.String())
	}
	if n := strings.Count(shown.String(), "branch 'main' already exists"); n != 1 {
		t.Fatalf("error shown %d times: %q", n, shown.String())
	}
}

func takenName(s string) string {
	switch strings.TrimSpace(s) {
	case "":
		return "branch name is required"
	case "main":
		return "branch 'main' already exists"
	default:
		return ""
	}
}

func runValidatedLoop(t *testing.T, input string, check func(string) string) (string, string, error) {
	t.Helper()
	return runValidatedLoopFrom(t, "", input, check)
}

func runValidatedLoopFrom(t *testing.T, initial, input string, check func(string) string) (string, string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	go func() {
		_, _ = w.Write([]byte(input))
		_ = w.Close()
	}()
	var out bytes.Buffer
	got, err := NewTTY(r, &out).validatedLineLoop(r, "Branch name", initial, check)
	return got, out.String(), err
}

func readScriptedKey(t *testing.T, input []byte, textLine bool) pickKey {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	go func() {
		_, _ = w.Write(input)
		_ = w.Close()
	}()
	var key pickKey
	if textLine {
		key, err = readValidatedKey(r)
	} else {
		key, err = readPickKey(r)
	}
	if err != nil {
		t.Fatal(err)
	}
	return key
}

type validatedStub struct {
	answers []string
	labels  []string
	n       int
}

func (p *validatedStub) String(label, _ string) (string, error) {
	p.labels = append(p.labels, label)
	if p.n >= len(p.answers) {
		return "", io.EOF
	}
	ans := p.answers[p.n]
	p.n++
	return ans, nil
}

func (p *validatedStub) Confirm(string, bool) (bool, error) { return false, nil }
func (p *validatedStub) Choose(string, []string) (int, error) {
	return -1, errors.New("unexpected choose")
}
func (p *validatedStub) Pick(string, []Choice, string) (string, error) {
	return "", errors.New("unexpected pick")
}
func (p *validatedStub) Required(string, string) error { return nil }
func (p *validatedStub) EditOrAccept(string, string) (string, error) {
	return "", errors.New("unexpected edit")
}
func (p *validatedStub) Optional(string, string) (string, error) {
	return "", errors.New("unexpected optional")
}
func (p *validatedStub) Closed(string, []string, string, bool) (string, error) {
	return "", errors.New("unexpected closed")
}
func (p *validatedStub) BatchChoice(string, string) (BatchDecision, error) {
	return BatchReject, nil
}
