package prompt

import "strings"

const (
	enterAccept = "press enter to accept"
	enterSkip   = "press enter to skip"
)

var (
	yesNoOptions       = []string{"yes", "no"}
	batchChoiceOptions = []string{"yes", "no", "all", "skip"}
)

// RequiredLine formats a required text question (see docs/reference/interaction.md).
// With a suggestion: `<prompt> [value] (press enter to accept):`
// Without: `<prompt>:`
func RequiredLine(prompt, suggestion string) string {
	action := ""
	if suggestion != "" {
		action = enterAccept
	}
	return questionLine(prompt, suggestion, action)
}

// OptionalLine formats an optional text question (see docs/reference/interaction.md).
// With a suggestion: `<prompt> [value] (press enter to accept):`.
// Without: `<prompt> (press enter to skip):`.
func OptionalLine(prompt, suggestion string) string {
	if suggestion != "" {
		return RequiredLine(prompt, suggestion)
	}
	return questionLine(prompt, "", enterSkip)
}

// ClosedLine formats a closed-list question (see docs/reference/interaction.md).
// Options appear as `[a/b/c]`. A default adds `(press enter to '<word>')`.
func ClosedLine(prompt string, options []string, defaultWord string) string {
	suggested := strings.Join(options, "/")
	action := ""
	if defaultWord != "" {
		action = "press enter to '" + defaultWord + "'"
	}
	return questionLine(prompt, suggested, action)
}

func questionLine(prompt, suggested, enterAction string) string {
	parts := []string{prompt}
	if suggested != "" {
		parts = append(parts, "["+suggested+"]")
	}
	if enterAction != "" {
		parts = append(parts, "("+enterAction+")")
	}
	return strings.Join(parts, " ") + ":"
}

// MatchClosed matches input against a closed list (full word or unique first
// letter, case-insensitive). Empty input takes defaultWord when it is one of
// the options; otherwise it does not match.
func MatchClosed(input string, options []string, defaultWord string) (string, bool) {
	s := strings.TrimSpace(input)
	if s == "" {
		return matchClosedDefault(options, defaultWord)
	}
	var word string
	words := 0
	for _, opt := range options {
		if strings.EqualFold(opt, s) {
			word = opt
			words++
		}
	}
	if words == 1 {
		return word, true
	}
	if len(s) != 1 {
		return "", false
	}
	letter := strings.ToLower(s)
	var hit string
	hits := 0
	for _, opt := range options {
		if opt == "" {
			continue
		}
		first := strings.ToLower(opt[:1])
		if first == letter {
			hit = opt
			hits++
		}
	}
	if hits == 1 {
		return hit, true
	}
	return "", false
}

func matchClosedDefault(options []string, defaultWord string) (string, bool) {
	if defaultWord == "" {
		return "", false
	}
	for _, opt := range options {
		if strings.EqualFold(opt, defaultWord) {
			return opt, true
		}
	}
	return "", false
}
