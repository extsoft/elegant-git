// Package text provides colored terminal output helpers for CLI messages.
package text

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	formatNormal = 0
	formatBold   = 1
	colorRed     = 31
	colorGreen   = 32
	colorBlue    = 34
	colorPurple  = 35
)

var out io.Writer = os.Stdout

// SetOutput redirects styled messages (for tests).
func SetOutput(w io.Writer) {
	out = w
}

func isTTY() bool {
	f, ok := out.(*os.File)
	if !ok {
		return false
	}
	return f == os.Stdout && isStdoutTTY()
}

func isStdoutTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func coloredLine(format, color int, parts ...string) {
	msg := strings.Join(parts, " ")
	if isTTY() {
		fmt.Fprintf(out, "\x1b[%d;%dm%s\x1b[m", format, color, msg)
		return
	}
	fmt.Fprint(out, msg)
}

func coloredText(format, color int, parts ...string) {
	coloredLine(format, color, parts...)
	fmt.Fprintln(out)
}

// CommandText prints a CLI command with the ==>> prefix.
func CommandText(parts ...string) {
	if isTTY() {
		fmt.Fprintf(out, "\x1b[%d;%dm==>> \x1b[m", formatBold, colorGreen)
	} else {
		fmt.Fprint(out, "==>> ")
	}
	coloredText(formatBold, colorBlue, parts...)
}

// PlainText prints an unstyled message (terminal default color).
func PlainText(parts ...string) {
	fmt.Fprintln(out, strings.Join(parts, " "))
}

// InfoText prints a regular informational message.
func InfoText(parts ...string) {
	coloredText(formatNormal, colorGreen, parts...)
}

// ErrorText prints an error message on one line.
func ErrorText(parts ...string) {
	coloredText(formatBold, colorRed, parts...)
}

// QuestionText prints a question without a trailing newline.
func QuestionText(parts ...string) {
	coloredLine(formatNormal, colorPurple, parts...)
}

func boxText(layout func(...string), parts ...string) {
	msg := strings.Join(parts, " ")
	border := strings.Repeat("=", len(msg)+4)
	layout(border)
	layout("==", msg, "==")
	layout(border)
}

// InfoBox prints an important message in a box.
func InfoBox(parts ...string) {
	boxText(func(p ...string) { InfoText(p...) }, parts...)
}

// Complete prints a final confirmation after a command finishes successfully.
func Complete(parts ...string) {
	InfoBox(parts...)
}

// ErrorBox prints an error message in a box.
func ErrorBox(parts ...string) {
	boxText(func(p ...string) { ErrorText(p...) }, parts...)
}

// OrUnset returns v, or "(unset)" when v is empty.
func OrUnset(v string) string {
	if v == "" {
		return "(unset)"
	}
	return v
}
