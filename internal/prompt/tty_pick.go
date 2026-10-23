package prompt

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/extsoft/elegant-git/internal/text"
	"golang.org/x/term"
)

const pickMinChoices = 2

func (t *TTY) Pick(label string, choices []Choice, defaultWord string) (string, error) {
	if len(choices) == 0 {
		return "", fmt.Errorf("no choices for %s", label)
	}
	choices = normalizeChoices(choices)
	if len(choices) < pickMinChoices {
		return t.pickAsClosed(label, choices, defaultWord)
	}
	in, ok := t.in.(*os.File)
	if !ok || !term.IsTerminal(int(in.Fd())) {
		return t.pickFallback(label, choices, defaultWord)
	}
	return t.pickInline(in, label, choices, defaultWord)
}

func normalizeChoices(choices []Choice) []Choice {
	out := make([]Choice, len(choices))
	for i, c := range choices {
		out[i] = Choice{Value: c.Value, Display: c.Display, Description: truncateDesc(c.Description, descMaxLen)}
	}
	return out
}

func (t *TTY) pickAsClosed(label string, choices []Choice, defaultWord string) (string, error) {
	opts := make([]string, len(choices))
	for i, c := range choices {
		opts[i] = c.Value
	}
	return t.askClosed(label, opts, defaultWord, true)
}

func pickSelectedValue(line string) string {
	if i := strings.IndexByte(line, '\t'); i >= 0 {
		return line[:i]
	}
	return line
}

func (t *TTY) pickFallback(label string, choices []Choice, defaultWord string) (string, error) {
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
	if len(matches) < pickMinChoices {
		return t.pickAsClosed(label, matches, defaultWord)
	}
	opts := make([]string, len(matches))
	for i, c := range matches {
		opts[i] = c.Value
	}
	return t.askClosed(label, opts, defaultWord, true)
}

func (t *TTY) pickInline(in *os.File, label string, choices []Choice, defaultWord string) (string, error) {
	oldState, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return t.pickFallback(label, choices, defaultWord)
	}
	defer func() { _ = term.Restore(int(in.Fd()), oldState) }()
	// Long prompts + placeholder wrap; logical-line CUU then reprints a new copy.
	fmt.Fprint(t.out, "\033[?7l")
	defer fmt.Fprint(t.out, "\033[?7h")
	flushWriter(t.out)

	filter := ""
	cursor := indexOfChoice(choices, defaultWord)
	prevLines := 0

	for {
		matches := filterChoices(choices, filter)
		if len(matches) == 0 {
			cursor = 0
		} else if cursor >= len(matches) {
			cursor = len(matches) - 1
		}

		visible, sel := visibleChoices(matches, cursor)
		lines, cursorCol := buildPickScreen(pickScreenInput{
			Label:      label,
			Filter:     filter,
			Visible:    visible,
			Sel:        sel,
			ValueWidth: valueColumnWidth(matches),
			Filtered:   len(matches),
			Total:      len(choices),
		})
		writePickScreen(t.out, lines, prevLines, label, filter, pickPlaceholderSingle, cursorCol)
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
			selected := matches[cursor]
			finalizePickScreen(t.out, label, choiceLabel(selected), prevLines)
			return selected.Value, nil
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

func indexOfChoice(choices []Choice, value string) int {
	if value == "" {
		return 0
	}
	for i, c := range choices {
		if c.Value == value {
			return i
		}
	}
	return 0
}

type pickScreenInput struct {
	Label       string
	Filter      string
	Visible     []Choice
	Sel         int
	ValueWidth  int
	Filtered    int
	Total       int
	Multi       bool
	Selected    int // multi-select count; omitted from status when Multi is false
	SelectedSet map[string]bool
}

func buildPickScreen(in pickScreenInput) (lines []string, cursorCol int) {
	prompt := in.Label + ": " + in.Filter
	cursorCol = len(prompt) + 1
	lines = []string{prompt, pickStatusLine(in)}
	if len(in.Visible) == 0 {
		blank := "  "
		if in.Multi {
			blank = "   "
		}
		lines = append(lines, blank+"(no matches)")
	} else {
		for i, c := range in.Visible {
			selected := in.Multi && in.SelectedSet[c.Value]
			lines = append(lines, formatPickLine(i == in.Sel, selected, in.Multi, c, in.ValueWidth, in.Filter))
		}
	}
	return lines, cursorCol
}

func pickStatusLine(in pickScreenInput) string {
	status := fmt.Sprintf("%d of %d", in.Filtered, in.Total)
	if in.Multi {
		status += fmt.Sprintf(" (%d selected)", in.Selected)
	}
	return status + " " + strings.Repeat("─", pickRuleWidth)
}

func paintPickPrompt(line, label, filter, placeholder string) string {
	prefix := label + ": "
	if !strings.HasPrefix(line, prefix) {
		return line
	}
	out := "\033[1;34m" + prefix + "\033[m"
	if filter != "" {
		return out + line[len(prefix):]
	}
	if placeholder == "" {
		placeholder = pickPlaceholderSingle
	}
	return out + "\033[3m" + placeholder + "\033[m"
}

func writePickScreen(out io.Writer, lines []string, prevLines int, label, filter, placeholder string, cursorCol int) {
	// Cursor is on the prompt row. Erase from here down so wrapped leftovers
	// from a previous frame cannot stack. Wrap is off during pickInline.
	fmt.Fprint(out, "\r\033[J")
	for i, line := range lines {
		painted := line
		if i == 0 {
			painted = paintPickPrompt(line, label, filter, placeholder)
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

func erasePickBlock(out io.Writer, lines int) {
	if lines <= 0 {
		return
	}
	for i := 0; i < lines; i++ {
		fmt.Fprint(out, "\r\033[2K")
		if i+1 < lines {
			fmt.Fprint(out, "\r\n")
		}
	}
	if lines > 1 {
		fmt.Fprintf(out, "\033[%dA", lines-1)
	}
	fmt.Fprint(out, "\r")
}

func finalizePickScreen(out io.Writer, label, value string, prevLines int) {
	erasePickBlock(out, prevLines)
	fmt.Fprintf(out, "\r\033[1;34m%s: \033[m%s\r\n", label, value)
	flushWriter(out)
}

func clearPickScreen(out io.Writer, lines int) {
	erasePickBlock(out, lines)
	fmt.Fprint(out, "\r\033[2K\r\n")
	flushWriter(out)
}

func flushWriter(out io.Writer) {
	type flusher interface{ Flush() error }
	if f, ok := out.(flusher); ok {
		_ = f.Flush()
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

func (t *TTY) PickMany(label string, choices []Choice, selected []string) ([]string, error) {
	if len(choices) == 0 {
		return nil, fmt.Errorf("no choices for %s", label)
	}
	choices = normalizeChoices(choices)
	in, ok := t.in.(*os.File)
	if !ok || !term.IsTerminal(int(in.Fd())) {
		return t.pickManyFallback(label, choices, selected)
	}
	return t.pickManyInline(in, label, choices, selected)
}

func (t *TTY) pickManyFallback(_ string, choices []Choice, selected []string) ([]string, error) {
	sel := selectionSet(choices, selected)
	width := valueColumnWidth(choices)
	for _, c := range choices {
		fmt.Fprintln(t.out, formatPickLine(false, sel[c.Value], true, c, width, ""))
	}
	for {
		line, err := t.Optional("Toggle", "")
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return selectedValues(choices, sel), nil
		}
		toggles, ok := toggleTokens(choices, line)
		if !ok {
			text.ErrorText(fmt.Sprintf("%q is not a choice", line))
			continue
		}
		for _, v := range toggles {
			sel[v] = !sel[v]
		}
		return selectedValues(choices, sel), nil
	}
}

func (t *TTY) pickManyInline(in *os.File, label string, choices []Choice, selected []string) ([]string, error) {
	oldState, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return t.pickManyFallback(label, choices, selected)
	}
	defer func() { _ = term.Restore(int(in.Fd()), oldState) }()
	fmt.Fprint(t.out, "\033[?7l")
	defer fmt.Fprint(t.out, "\033[?7h")
	flushWriter(t.out)

	sel := selectionSet(choices, selected)
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

		visible, visSel := visibleChoices(matches, cursor)
		lines, cursorCol := buildPickScreen(pickScreenInput{
			Label:       label,
			Filter:      filter,
			Visible:     visible,
			Sel:         visSel,
			ValueWidth:  valueColumnWidth(matches),
			Filtered:    len(matches),
			Total:       len(choices),
			Multi:       true,
			Selected:    countSelected(sel),
			SelectedSet: sel,
		})
		writePickScreen(t.out, lines, prevLines, label, filter, pickPlaceholderMulti, cursorCol)
		prevLines = len(lines)

		key, err := readPickKey(in)
		if err != nil {
			return nil, err
		}
		switch key.kind {
		case pickKeyCancel:
			clearPickScreen(t.out, prevLines)
			return nil, ErrUserCancelled
		case pickKeyEnter:
			finalizePickScreen(t.out, label, selectedSummary(choices, sel), prevLines)
			return selectedValues(choices, sel), nil
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
			if key.rune == ' ' {
				if len(matches) > 0 {
					v := matches[cursor].Value
					sel[v] = !sel[v]
				}
				continue
			}
			filter += string(key.rune)
			cursor = 0
		}
	}
}

func selectionSet(choices []Choice, selected []string) map[string]bool {
	allowed := make(map[string]bool, len(choices))
	for _, c := range choices {
		allowed[c.Value] = true
	}
	sel := map[string]bool{}
	for _, v := range selected {
		if allowed[v] {
			sel[v] = true
		}
	}
	return sel
}

func selectedValues(choices []Choice, sel map[string]bool) []string {
	out := make([]string, 0, len(choices))
	for _, c := range choices {
		if sel[c.Value] {
			out = append(out, c.Value)
		}
	}
	return out
}

func selectedSummary(choices []Choice, sel map[string]bool) string {
	var labels []string
	for _, c := range choices {
		if sel[c.Value] {
			labels = append(labels, c.Value)
		}
	}
	if len(labels) == 0 {
		return "(none)"
	}
	return strings.Join(labels, ", ")
}

func countSelected(sel map[string]bool) int {
	n := 0
	for _, on := range sel {
		if on {
			n++
		}
	}
	return n
}

func toggleTokens(choices []Choice, line string) ([]string, bool) {
	if _, ok := choiceByValue(choices, line); ok {
		return []string{line}, true
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil, true
	}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if _, ok := choiceByValue(choices, field); !ok {
			return nil, false
		}
		out = append(out, field)
	}
	return out, true
}

func choiceByValue(choices []Choice, value string) (string, bool) {
	for _, c := range choices {
		if c.Value == value {
			return c.Value, true
		}
	}
	return "", false
}
