package prompt

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/bees-hive/elegant-git/internal/text"
	"golang.org/x/term"
)

const (
	pickFzfHeight = "40%"
	pickFzfLayout = "reverse"
)

func (t *TTY) Pick(label string, choices []Choice) (string, error) {
	if len(choices) == 0 {
		return "", fmt.Errorf("no choices for %s", label)
	}
	in, ok := t.in.(*os.File)
	if !ok || !term.IsTerminal(int(in.Fd())) {
		return t.pickFallback(label, choices)
	}
	if fzfPath, err := exec.LookPath("fzf"); err == nil {
		if val, err := t.pickFzf(fzfPath, label, choices); err == nil || err == ErrUserCancelled {
			return val, err
		}
	}
	return t.pickInline(in, label, choices)
}

func (t *TTY) pickFzf(fzfPath, label string, choices []Choice) (string, error) {
	var stdin bytes.Buffer
	for _, c := range choices {
		if c.Description != "" {
			fmt.Fprintf(&stdin, "%s\t%s\n", c.Value, c.Description)
		} else {
			fmt.Fprintln(&stdin, c.Value)
		}
	}

	args := []string{
		"--height", pickFzfHeight,
		"--layout", pickFzfLayout,
		"--border",
		"--prompt", label + "> ",
		"--no-info",
		"--delimiter", "\t",
		"--with-nth", "1",
	}

	cmd := exec.Command(fzfPath, args...)
	cmd.Stdin = &stdin
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			switch ee.ExitCode() {
			case 130, 1:
				return "", ErrUserCancelled
			}
		}
		return "", err
	}

	selected := strings.TrimSpace(stdout.String())
	if selected == "" {
		return "", ErrUserCancelled
	}
	return pickSelectedValue(selected), nil
}

func pickSelectedValue(line string) string {
	if i := strings.IndexByte(line, '\t'); i >= 0 {
		return line[:i]
	}
	return line
}

func (t *TTY) pickFallback(label string, choices []Choice) (string, error) {
	text.QuestionText(RequiredLine(label, "") + " ")
	filter, err := t.reader().ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	filter = strings.TrimSpace(filter)
	matches := filterChoices(choices, filter)
	if len(matches) == 0 {
		return "", fmt.Errorf("no match for %q", filter)
	}
	if len(matches) == 1 {
		return matches[0].Value, nil
	}
	opts := make([]string, len(matches))
	for i, c := range matches {
		opts[i] = c.Value
	}
	return t.askClosed(label, opts, "", true)
}

func (t *TTY) pickInline(in *os.File, label string, choices []Choice) (string, error) {
	oldState, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return t.pickFallback(label, choices)
	}
	defer func() { _ = term.Restore(int(in.Fd()), oldState) }()

	filter := ""
	cursor := 0
	prevLines := 0

	for {
		matches := filterChoices(choices, filter)
		if len(matches) == 0 {
			cursor = 0
		} else if cursor >= len(matches) {
			cursor = len(matches) - 1
		}

		visible, sel := visibleChoices(matches, cursor)
		lines := buildPickScreen(label, filter, visible, sel)
		writePickScreen(t.out, lines, prevLines)
		prevLines = len(lines)

		key, err := readPickKey(in)
		if err != nil {
			return "", err
		}
		switch key.kind {
		case pickKeyCancel:
			clearPickScreen(t.out, prevLines)
			return "", ErrUserCancelled
		case pickKeyEnter:
			if len(matches) == 0 {
				continue
			}
			selected := matches[cursor].Value
			finalizePickScreen(t.out, label, selected, prevLines)
			return selected, nil
		case pickKeyUp:
			if cursor > 0 {
				cursor--
			}
		case pickKeyDown:
			if cursor < len(matches)-1 {
				cursor++
			}
		case pickKeyBackspace:
			if filter != "" {
				filter = filter[:len(filter)-1]
				cursor = 0
			}
		case pickKeyRune:
			filter += string(key.rune)
			cursor = 0
		}
	}
}

func buildPickScreen(label, filter string, visible []Choice, sel int) []string {
	prompt := label + "> "
	if filter == "" {
		prompt += "_"
	} else {
		prompt += filter
	}
	lines := []string{prompt}
	if len(visible) == 0 {
		lines = append(lines, "  (no matches)")
	} else {
		for i, c := range visible {
			lines = append(lines, formatPickLine(i == sel, c))
		}
	}
	return lines
}

func writePickScreen(out io.Writer, lines []string, prevLines int) {
	if prevLines > 0 {
		fmt.Fprint(out, "\033[", prevLines, "A")
	}
	for i, line := range lines {
		if i < prevLines {
			fmt.Fprint(out, "\r\033[K", line)
		} else {
			fmt.Fprintln(out, line)
		}
	}
	if prevLines > len(lines) {
		for i := len(lines); i < prevLines; i++ {
			fmt.Fprint(out, "\r\033[K")
			if i+1 < prevLines {
				fmt.Fprintln(out)
			}
		}
	}
}

func finalizePickScreen(out io.Writer, label, value string, prevLines int) {
	if prevLines == 0 {
		fmt.Fprintf(out, "%s> %s\n", label, value)
		return
	}
	fmt.Fprint(out, "\033[", prevLines, "A")
	for i := 0; i < prevLines; i++ {
		fmt.Fprint(out, "\r\033[K")
		if i+1 < prevLines {
			fmt.Fprintln(out)
		}
	}
	fmt.Fprintf(out, "%s> %s\n", label, value)
}

func clearPickScreen(out io.Writer, lines int) {
	if lines == 0 {
		return
	}
	fmt.Fprint(out, "\033[", lines, "A")
	for i := 0; i < lines; i++ {
		fmt.Fprint(out, "\r\033[K")
		if i+1 < lines {
			fmt.Fprintln(out)
		}
	}
}

type pickKeyKind int

const (
	pickKeyNone pickKeyKind = iota
	pickKeyEnter
	pickKeyCancel
	pickKeyUp
	pickKeyDown
	pickKeyBackspace
	pickKeyRune
)

type pickKey struct {
	kind pickKeyKind
	rune rune
}

func readPickKey(in *os.File) (pickKey, error) {
	var buf [3]byte
	n, err := in.Read(buf[:1])
	if err != nil {
		return pickKey{}, err
	}
	if n == 0 {
		return pickKey{}, io.EOF
	}
	b := buf[0]
	switch b {
	case 3, 4: // Ctrl+C, Ctrl+D
		return pickKey{kind: pickKeyCancel}, nil
	case 13, 10:
		return pickKey{kind: pickKeyEnter}, nil
	case 27:
		n2, err := in.Read(buf[1:3])
		if err != nil || n2 == 0 {
			return pickKey{kind: pickKeyCancel}, nil
		}
		if n2 >= 2 && buf[1] == '[' {
			switch buf[2] {
			case 'A':
				return pickKey{kind: pickKeyUp}, nil
			case 'B':
				return pickKey{kind: pickKeyDown}, nil
			}
		}
		return pickKey{kind: pickKeyCancel}, nil
	case 127, 8:
		return pickKey{kind: pickKeyBackspace}, nil
	default:
		if isPrintable(rune(b)) {
			return pickKey{kind: pickKeyRune, rune: rune(b)}, nil
		}
		return pickKey{kind: pickKeyNone}, nil
	}
}
