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
	prompt := question
	if defaultVal != "" {
		prompt = question + " {" + defaultVal + "}"
	}
	text.QuestionText(prompt + ": ")
	line, err := t.reader().ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal, nil
	}
	return line, nil
}

func (t *TTY) Confirm(question string) (bool, error) {
	text.QuestionText(question + " [y/N]: ")
	line, err := t.reader().ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes", nil
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
	prompt := label
	if suggested != "" {
		prompt = label + " {" + suggested + "}"
	}
	text.QuestionText(prompt + ": ")
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

func (t *TTY) BatchChoice(question string) (BatchDecision, error) {
	text.QuestionText(question + " [y/n/A/S]: ")
	line, err := t.reader().ReadString('\n')
	if err != nil && err != io.EOF {
		return BatchReject, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return BatchConfirm, nil
	case "a", "all":
		return BatchApplyAll, nil
	case "s", "skip":
		return BatchSkip, nil
	default:
		return BatchReject, nil
	}
}
