package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/bees-hive/elegant-git/internal/text"
)

// TTY implements Prompter using stdin and stdout.
type TTY struct {
	in  io.Reader
	out io.Writer
	br  *bufio.Reader
}

// NewTTY returns a prompter; nil in/out default to os.Stdin/os.Stdout.
func NewTTY(in io.Reader, out io.Writer) *TTY {
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	return &TTY{in: in, out: out}
}

func (t *TTY) reader() *bufio.Reader {
	if t.br == nil {
		t.br = bufio.NewReader(t.in)
	}
	return t.br
}

func (t *TTY) String(question, defaultVal string) (string, error) {
	q := RequiredLine(question, defaultVal) + " "
	for {
		text.QuestionText(q)
		line, err := t.reader().ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		line = strings.TrimSpace(line)
		if line != "" {
			return line, nil
		}
		if defaultVal != "" {
			return defaultVal, nil
		}
		if err == io.EOF {
			return "", err
		}
	}
}

func (t *TTY) Confirm(question string, defaultYes bool) (bool, error) {
	def := "no"
	if defaultYes {
		def = "yes"
	}
	ans, err := t.askClosed(question, yesNoOptions, def, true)
	if err != nil {
		return false, err
	}
	return ans == "yes", nil
}

func (t *TTY) Choose(question string, options []string) (int, error) {
	if len(options) == 0 {
		return -1, fmt.Errorf("no options to choose from")
	}
	fmt.Fprintln(t.out)
	fmt.Fprintln(t.out, question)
	for i, opt := range options {
		fmt.Fprintf(t.out, "  %d) %s\n", i+1, opt)
	}
	text.QuestionText("Enter choice (number): ")
	line, err := t.reader().ReadString('\n')
	if err != nil && err != io.EOF {
		return -1, err
	}
	line = strings.TrimSpace(line)
	n, err := strconv.Atoi(line)
	if err != nil || n < 1 || n > len(options) {
		return -1, fmt.Errorf("invalid choice: %q", line)
	}
	return n - 1, nil
}

func (t *TTY) Required(label, current string) error {
	if strings.TrimSpace(current) != "" {
		return nil
	}
	_, err := t.String(label, "")
	if err != nil {
		return err
	}
	return nil
}

func (t *TTY) EditOrAccept(label, suggested string) (string, error) {
	q := RequiredLine(label, suggested)
	if suggested == "" {
		q = OptionalLine(label, "")
	}
	text.QuestionText(q + " ")
	line, err := t.reader().ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line != "" {
		return line, nil
	}
	if suggested != "" {
		return suggested, nil
	}
	return "", nil
}

func (t *TTY) Optional(label, suggested string) (string, error) {
	text.QuestionText(OptionalLine(label, suggested) + " ")
	line, err := t.reader().ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (t *TTY) Closed(question string, options []string, defaultWord string, required bool) (string, error) {
	return t.askClosed(question, options, defaultWord, required)
}

func (t *TTY) BatchChoice(question, defaultWord string) (BatchDecision, error) {
	ans, err := t.askClosed(question, batchChoiceOptions, defaultWord, true)
	if err != nil {
		return BatchReject, err
	}
	switch ans {
	case "yes":
		return BatchConfirm, nil
	case "all":
		return BatchApplyAll, nil
	case "skip":
		return BatchSkip, nil
	default:
		return BatchReject, nil
	}
}

func (t *TTY) askClosed(question string, options []string, defaultWord string, required bool) (string, error) {
	q := ClosedLine(question, options, defaultWord) + " "
	for {
		text.QuestionText(q)
		line, err := t.reader().ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		if got, ok := MatchClosed(line, options, defaultWord); ok {
			return got, nil
		}
		if !required && strings.TrimSpace(line) == "" {
			return "", nil
		}
		if err == io.EOF {
			return "", err
		}
	}
}
