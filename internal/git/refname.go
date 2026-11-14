package git

import (
	"fmt"
	"strings"
	"unicode"
)

// CheckBranchName reports whether name is a valid git branch name.
func CheckBranchName(name string) error {
	if err := validateBranchName(name); err != nil {
		return err
	}
	out, err := Output("check-ref-format", "--branch", name)
	return ErrorFromOutput(out, err)
}

// ErrorFromOutput returns err, using git's own text when the command wrote any.
func ErrorFromOutput(out string, err error) error {
	if err == nil {
		return nil
	}
	if msg := strings.TrimSpace(out); msg != "" {
		return fmt.Errorf("%s", msg)
	}
	return err
}

// NewBranchNameProblem reports why name cannot be created as a local branch.
// branches are existing short names. An empty result means the name is usable.
// Comparison is case-sensitive; a slash boundary with an existing name conflicts.
func NewBranchNameProblem(name string, branches []string) string {
	if err := validateBranchName(name); err != nil {
		return err.Error()
	}
	for _, existing := range branches {
		if existing == name {
			return fmt.Sprintf("branch '%s' already exists", name)
		}
	}
	for _, existing := range branches {
		if branchNamesConflict(name, existing) {
			return fmt.Sprintf("branch '%s' conflicts with existing branch '%s'", name, existing)
		}
	}
	return ""
}

func branchNamesConflict(name, existing string) bool {
	if existing == "" || existing == name {
		return false
	}
	return strings.HasPrefix(name, existing+"/") || strings.HasPrefix(existing, name+"/")
}

func validateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name is required")
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("branch name cannot start with '.'")
	}
	if strings.HasSuffix(name, ".") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("branch name cannot end with '.' or '/'")
	}
	if strings.HasSuffix(name, ".lock") {
		return fmt.Errorf("branch name cannot end with '.lock'")
	}
	if strings.Contains(name, "..") || strings.Contains(name, "//") {
		return fmt.Errorf("branch name cannot contain '..' or '//'")
	}
	if strings.Contains(name, "@{") {
		return fmt.Errorf("branch name cannot contain '@{'")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("branch name cannot contain control characters")
		}
		switch r {
		case ' ', '~', '^', ':', '?', '*', '[', '\\':
			return fmt.Errorf("branch name cannot contain %q", r)
		}
	}
	return nil
}
