package prompt

import (
	"fmt"
	"io"
	"os"
	"unicode"
	"unicode/utf8"

	"github.com/extsoft/elegant-git/internal/text"
	"golang.org/x/term"
)

// LineValidator is the optional prompter surface for a line checked while typing.
type LineValidator interface {
	ValidatedLine(label, initial string, check func(string) string) (string, error)
}

// ValidatedLine asks for one line, starting from initial. A LineValidator shows
// check's message while the line is edited and ignores Enter until check returns
// empty. An untouched empty line shows no problem. Any other prompter prints
// that message and asks again.
func ValidatedLine(p Prompter, label, initial string, check func(string) string) (string, error) {
	check = normalizeLineCheck(check)
	if v, ok := p.(LineValidator); ok {
		return v.ValidatedLine(label, initial, check)
	}
	return validatedFallback(p, label, initial, check)
}

func (t *TTY) ValidatedLine(label, initial string, check func(string) string) (string, error) {
	check = normalizeLineCheck(check)
	in, ok := t.in.(*os.File)
	if !ok || !term.IsTerminal(int(in.Fd())) {
		return validatedFallback(t, label, initial, check)
	}
	return t.validatedLineRaw(in, label, initial, check)
}

func (t *TTY) validatedLineRaw(in *os.File, label, initial string, check func(string) string) (string, error) {
	oldState, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return validatedFallback(t, label, initial, check)
	}
	defer func() { _ = term.Restore(int(in.Fd()), oldState) }()
	fmt.Fprint(t.out, "\033[?7l")
	defer fmt.Fprint(t.out, "\033[?7h")
	flushWriter(t.out)
	return t.validatedLineLoop(in, label, initial, check)
}

func (t *TTY) validatedLineLoop(in *os.File, label, initial string, check func(string) string) (string, error) {
	value := initial
	dirty := false
	prev := 0
	for {
		problem := check(value)
		shown := problem
		if value == "" && !dirty {
			shown = ""
		}
		lines, col := buildValidatedScreen(label, value, shown)
		writeValidatedScreen(t.out, lines, prev, label, value, shown, col)
		prev = len(lines)

		key, err := readValidatedKey(in)
		if err != nil {
			return "", err
		}
		switch key.kind {
		case pickKeyCancel:
			clearPickScreen(t.out, prev)
			return "", ErrUserCancelled
		case pickKeyEnter:
			if problem != "" {
				continue
			}
			finalizePickScreen(t.out, label, value, prev)
			return value, nil
		case pickKeyBackspace:
			value = dropLastRune(value)
			dirty = true
		case pickKeyWordBackspace:
			value = dropLastWord(value)
			dirty = true
		case pickKeyRune:
			value += string(key.rune)
			dirty = true
		}
	}
}

func buildValidatedScreen(label, value, problem string) (lines []string, cursorCol int) {
	line := label + ": " + value
	lines = []string{line}
	if problem != "" {
		lines = append(lines, problem)
	}
	return lines, displayWidth(line) + 1
}

func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

func runeWidth(r rune) int {
	if r < 0x20 || r == 0x7f || r == 0x200D || r == 0xFE0E || r == 0xFE0F {
		return 0
	}
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
		return 0
	}
	if isWideRune(r) {
		return 2
	}
	return 1
}

func isWideRune(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F:
		return true
	case r >= 0x2329 && r <= 0x232A:
		return true
	case r >= 0x2E80 && r <= 0xA4CF && r != 0x303F:
		return true
	case r >= 0xAC00 && r <= 0xD7A3:
		return true
	case r >= 0xF900 && r <= 0xFAFF:
		return true
	case r >= 0xFE10 && r <= 0xFE19:
		return true
	case r >= 0xFE30 && r <= 0xFE6F:
		return true
	case r >= 0xFF00 && r <= 0xFF60:
		return true
	case r >= 0xFFE0 && r <= 0xFFE6:
		return true
	case r >= 0x1F300 && r <= 0x1FAFF:
		return true
	case r >= 0x20000 && r <= 0x3FFFD:
		return true
	default:
		return false
	}
}

func paintValidatedPrompt(label, value string) string {
	return "\033[1;34m" + label + ": \033[m" + value
}

func paintValidatedProblem(problem string) string {
	if problem == "" {
		return ""
	}
	return "\033[1;31m" + problem + "\033[m"
}

func writeValidatedScreen(out io.Writer, lines []string, prevLines int, label, value, problem string, cursorCol int) {
	fmt.Fprint(out, "\r\033[J")
	for i, line := range lines {
		painted := line
		switch i {
		case 0:
			painted = paintValidatedPrompt(label, value)
		case 1:
			painted = paintValidatedProblem(problem)
		}
		fmt.Fprint(out, painted, "\r\n")
	}
	extra := 0
	if prevLines > len(lines) {
		extra = prevLines - len(lines)
		for i := 0; i < extra; i++ {
			fmt.Fprint(out, "\r\n")
		}
	}
	if up := len(lines) + extra; up > 0 {
		fmt.Fprintf(out, "\033[%dA", up)
	}
	if cursorCol > 0 {
		fmt.Fprintf(out, "\033[%dG", cursorCol)
	}
	flushWriter(out)
}

func dropLastRune(s string) string {
	if s == "" {
		return s
	}
	_, size := utf8.DecodeLastRuneInString(s)
	if size <= 0 {
		return ""
	}
	return s[:len(s)-size]
}

func dropLastWord(s string) string {
	i := len(s)
	for i > 0 {
		r, size := utf8.DecodeLastRuneInString(s[:i])
		if !isWordSep(r) {
			break
		}
		i -= size
	}
	for i > 0 {
		r, size := utf8.DecodeLastRuneInString(s[:i])
		if isWordSep(r) {
			break
		}
		i -= size
	}
	return s[:i]
}

func isWordSep(r rune) bool {
	return unicode.IsSpace(r) || r == '/' || r == '-' || r == '_' || r == '.'
}

func normalizeLineCheck(check func(string) string) func(string) string {
	if check == nil {
		return func(string) string { return "" }
	}
	return check
}

func validatedFallback(p Prompter, label, initial string, check func(string) string) (string, error) {
	for {
		name, err := p.String(label, initial)
		if err != nil {
			return "", err
		}
		if problem := check(name); problem != "" {
			text.ErrorText(problem)
			// A line prompter returns its default on empty input or EOF, so
			// keeping a rejected value as the default would loop forever.
			initial = ""
			continue
		}
		return name, nil
	}
}
