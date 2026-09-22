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
	_, err := Output("check-ref-format", "--branch", name)
	return err
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
