package git

import (
	"fmt"
	"strings"
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
	if name == "HEAD" {
		return fmt.Errorf("branch name cannot be 'HEAD'")
	}
	if name[0] == '-' {
		return fmt.Errorf("branch name cannot start with '-'")
	}
	for i := 0; i < len(name); i++ {
		b := name[i]
		if b == ' ' {
			return fmt.Errorf("branch name cannot contain ' '")
		}
		if b <= ' ' || b == 0x7f {
			return fmt.Errorf("branch name cannot contain control characters")
		}
		switch b {
		case '~', '^', ':', '?', '*', '[', '\\':
			return fmt.Errorf("branch name cannot contain %q", rune(b))
		}
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("branch name cannot contain '..'")
	}
	if strings.Contains(name, "@{") {
		return fmt.Errorf("branch name cannot contain '@{'")
	}
	if strings.HasSuffix(name, ".") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("branch name cannot end with '.' or '/'")
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" {
			return fmt.Errorf("branch name cannot contain an empty component")
		}
		if strings.HasPrefix(part, ".") {
			return fmt.Errorf("branch name cannot contain a component starting with '.'")
		}
		if strings.HasSuffix(part, ".lock") {
			return fmt.Errorf("branch name cannot contain a component ending with '.lock'")
		}
	}
	return nil
}
