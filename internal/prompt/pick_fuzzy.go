package prompt

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	pickMaxVisible        = 10
	descMaxLen            = 70
	pickRuleWidth         = 20
	pickPlaceholderSingle = "select one option; enter to confirm"
	pickPlaceholderMulti  = "select one or more; space to select, enter to confirm"
)

func filterChoices(choices []Choice, query string) []Choice {
	query = strings.TrimSpace(query)
	if query == "" {
		return append([]Choice(nil), choices...)
	}
	type ranked struct {
		c           Choice
		tier, score int
	}
	var matches []ranked
	for _, c := range choices {
		if score, _, ok := fuzzyScore(query, choiceLabel(c)); ok {
			matches = append(matches, ranked{c, 0, score})
		} else if score, _, ok := fuzzyScore(query, c.Description); ok {
			matches = append(matches, ranked{c, 1, score})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].tier != matches[j].tier {
			return matches[i].tier < matches[j].tier
		}
		return matches[i].score > matches[j].score
	})
	out := make([]Choice, len(matches))
	for i, m := range matches {
		out[i] = m.c
	}
	return out
}

// fuzzyMatch reports whether query matches target as a case-insensitive subsequence.
func fuzzyMatch(query, target string) bool {
	_, _, ok := fuzzyScore(query, target)
	return ok
}

const (
	fuzzyBonusWordStart   = 8
	fuzzyBonusConsecutive = 4
	pickHitOn             = "\033[1;4m"
	pickHitOff            = "\033[m"
)

// fuzzyScore matches query against target as a case-insensitive subsequence.
// Higher scores favor word starts, consecutive runs, and early matches.
// Hits are byte offsets of the matched runes in target.
func fuzzyScore(query, target string) (int, []int, bool) {
	q := []rune(strings.ToLower(query))
	if len(q) == 0 {
		return 0, nil, true
	}
	var runes []rune
	var offsets []int
	for i, r := range target {
		runes = append(runes, r)
		offsets = append(offsets, i)
	}
	n, m := len(q), len(runes)
	if n > m {
		return 0, nil, false
	}
	score := make([][]int, n)
	prev := make([][]int, n)
	ok := make([][]bool, n)
	for i := range n {
		score[i] = make([]int, m)
		prev[i] = make([]int, m)
		ok[i] = make([]bool, m)
	}
	for j, r := range runes {
		if unicode.ToLower(r) != q[0] {
			continue
		}
		s := -j
		if j == 0 || isWordStart(runes[j-1], runes[j]) {
			s += fuzzyBonusWordStart
		}
		score[0][j] = s
		ok[0][j] = true
		prev[0][j] = -1
	}
	for i := 1; i < n; i++ {
		runVal, runP, runOK := 0, -1, false
		for j, r := range runes {
			if unicode.ToLower(r) == q[i] {
				best, bestP, found := 0, -1, false
				if j > 0 && ok[i-1][j-1] {
					best = score[i-1][j-1] + fuzzyBonusConsecutive
					bestP = j - 1
					found = true
				}
				if runOK {
					s := runVal - j + 1
					if !found || s > best {
						best, bestP, found = s, runP, true
					}
				}
				if found {
					if j == 0 || isWordStart(runes[j-1], runes[j]) {
						best += fuzzyBonusWordStart
					}
					score[i][j] = best
					prev[i][j] = bestP
					ok[i][j] = true
				}
			}
			if j > 0 && ok[i-1][j-1] {
				v := score[i-1][j-1] + (j - 1)
				if !runOK || v > runVal {
					runVal, runP, runOK = v, j-1, true
				}
			}
		}
	}
	bestJ, found := -1, false
	best := 0
	for j := range m {
		if ok[n-1][j] && (!found || score[n-1][j] > best) {
			best, bestJ, found = score[n-1][j], j, true
		}
	}
	if !found {
		return 0, nil, false
	}
	runeHits := make([]int, n)
	for i, j := n-1, bestJ; i >= 0; i-- {
		runeHits[i] = j
		j = prev[i][j]
	}
	hits := make([]int, n)
	for i, rh := range runeHits {
		hits[i] = offsets[rh]
	}
	return scoreHits(runes, runeHits), hits, true
}

func scoreHits(runes []rune, hits []int) int {
	score := -hits[0]
	for k, i := range hits {
		if i == 0 || isWordStart(runes[i-1], runes[i]) {
			score += fuzzyBonusWordStart
		}
		if k > 0 {
			if gap := i - hits[k-1] - 1; gap == 0 {
				score += fuzzyBonusConsecutive
			} else {
				score -= gap
			}
		}
	}
	return score
}

func isWordStart(prev, cur rune) bool {
	if !unicode.IsLetter(prev) && !unicode.IsDigit(prev) {
		return true
	}
	return unicode.IsLower(prev) && unicode.IsUpper(cur)
}

func highlightHits(s string, hits []int) string {
	if len(hits) == 0 {
		return s
	}
	var b strings.Builder
	on, next := false, 0
	for i := 0; i < len(s); {
		_, w := utf8.DecodeRuneInString(s[i:])
		hit := next < len(hits) && hits[next] == i
		if hit {
			next++
		}
		if hit != on {
			if hit {
				b.WriteString(pickHitOn)
			} else {
				b.WriteString(pickHitOff)
			}
			on = hit
		}
		b.WriteString(s[i : i+w])
		i += w
	}
	if on {
		b.WriteString(pickHitOff)
	}
	return b.String()
}

func truncateDesc(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func valueColumnWidth(choices []Choice) int {
	w := 0
	for _, c := range choices {
		if n := len(choiceLabel(c)); n > w {
			w = n
		}
	}
	return w
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func formatPickLine(current, selected bool, c Choice, valueWidth int, query string) string {
	var prefix string
	switch {
	case current && selected:
		prefix = ">*"
	case current:
		prefix = "> "
	case selected:
		prefix = "* "
	default:
		prefix = "  "
	}
	var labelHits, descHits []int
	if query = strings.TrimSpace(query); query != "" {
		if _, hits, ok := fuzzyScore(query, choiceLabel(c)); ok {
			labelHits = hits
		} else if _, hits, ok := fuzzyScore(query, c.Description); ok {
			descHits = hits
		}
	}
	line := prefix + highlightHits(padRight(choiceLabel(c), valueWidth), labelHits)
	if c.Description != "" {
		// Spaces for display columns; tabs expand unpredictably and break redraw.
		line += "  " + highlightHits(c.Description, descHits)
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
