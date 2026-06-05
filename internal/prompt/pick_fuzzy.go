package prompt

import (
	"strings"
	"unicode"
)

const pickMaxVisible = 6

func filterChoices(choices []Choice, query string) []Choice {
	query = strings.TrimSpace(query)
	if query == "" {
		return append([]Choice(nil), choices...)
	}
	var out []Choice
	for _, c := range choices {
		if fuzzyMatch(query, c.Value) || fuzzyMatch(query, c.Description) {
			out = append(out, c)
		}
	}
	return out
}

// fuzzyMatch reports whether query matches target as a case-insensitive subsequence.
func fuzzyMatch(query, target string) bool {
	if query == "" {
		return true
	}
	q := strings.ToLower(query)
	t := strings.ToLower(target)
	qi, ti := 0, 0
	for qi < len(q) && ti < len(t) {
		if q[qi] == t[ti] {
			qi++
		}
		ti++
	}
	return qi == len(q)
}

func formatPickLine(selected bool, c Choice) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}
	line := prefix + c.Value
	if c.Description != "" {
		line += "  " + c.Description
	}
	return line
}

func visibleChoices(matches []Choice, cursor int) ([]Choice, int) {
	if len(matches) == 0 {
		return nil, 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(matches) {
		cursor = len(matches) - 1
	}
	start := 0
	if len(matches) > pickMaxVisible {
		start = cursor - pickMaxVisible/2
		if start < 0 {
			start = 0
		}
		if start+pickMaxVisible > len(matches) {
			start = len(matches) - pickMaxVisible
		}
	}
	end := start + pickMaxVisible
	if end > len(matches) {
		end = len(matches)
	}
	visible := matches[start:end]
	return visible, cursor - start
}

func isPrintable(r rune) bool {
	return r >= 32 && r != 127 && !unicode.IsControl(r)
}
