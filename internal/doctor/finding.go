package doctor

import (
	"errors"
	"fmt"
	"io"

	"github.com/extsoft/elegant-git/internal/prompt"
)

// ErrSkipped means the user declined a Decide finding; Run treats it as no change.
var ErrSkipped = errors.New("skipped")

// Finding is one diagnosed problem and its proposed repair.
// A nil Apply means the finding is advisory: printed, never asked.
// Decide means Apply owns the choice (including ignore) and Run skips Fix?.
type Finding struct {
	Problem string
	Repair  string
	Apply   func() error
	Decide  bool
}

// Run prints findings and, interactively, confirms each repairable one.
// Non-interactive mode only prints; callers own the "N issue(s) found" error.
// Advisory findings (nil Apply) are printed and never confirmed.
func Run(w io.Writer, p prompt.Prompter, findings []Finding) (changed bool, err error) {
	if len(findings) == 0 {
		return false, nil
	}
	if prompt.NonInteractive(p) {
		printFindings(w, findings)
		return false, nil
	}
	for _, f := range findings {
		fmt.Fprintf(w, "problem: %s\n  repair: %s\n", f.Problem, f.Repair)
		if f.Apply == nil {
			continue
		}
		if !f.Decide {
			ok, err := p.Confirm("Fix?", true)
			if err != nil {
				return changed, err
			}
			if !ok {
				continue
			}
		}
		if err := f.Apply(); err != nil {
			if errors.Is(err, ErrSkipped) {
				continue
			}
			return changed, err
		}
		changed = true
	}
	return changed, nil
}

func printFindings(w io.Writer, findings []Finding) {
	for _, f := range findings {
		fmt.Fprintf(w, "problem: %s\n  repair: %s\n", f.Problem, f.Repair)
	}
}
