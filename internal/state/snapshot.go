package state

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/extsoft/elegant-git/internal/git"
)

// Snapshot is a read of the working tree. Errors from reading status are
// returned separately and are not treated as dirt.
type Snapshot struct {
	Bare      bool
	Unborn    bool
	Operation string
	Staged    []string
	Unstaged  []string
	Untracked []string
	Ignored   []string
	Unmerged  []string
}

// Relevant reports non-ignored uncommitted work.
func (s Snapshot) Relevant() bool {
	return len(s.Staged) > 0 || len(s.Unstaged) > 0 || len(s.Untracked) > 0 || len(s.Unmerged) > 0
}

// HasTracked reports staged, unstaged, or unmerged tracked edits.
func (s Snapshot) HasTracked() bool {
	return len(s.Staged) > 0 || len(s.Unstaged) > 0 || len(s.Unmerged) > 0
}

// StatusArgs is the NUL-delimited status read. The overrides see non-ignored
// untracked files even when the user's status settings hide them.
func StatusArgs() []string {
	return []string{
		"-c", "core.quotepath=false",
		"-c", "status.relativePaths=true",
		"-c", "status.showUntrackedFiles=all",
		"status",
		"--porcelain=v1",
		"-z",
		"--untracked-files=all",
		"--ignored",
		"--ignore-submodules=none",
		"--no-renames",
	}
}

// ReadSnapshot reads status without refreshing or changing the index.
func ReadSnapshot() (Snapshot, error) {
	s := Snapshot{
		Bare:      IsBareRepository(),
		Unborn:    HeadUnborn(),
		Operation: InProgressOperation(),
	}
	if s.Bare {
		return s, nil
	}
	out, err := git.Output(StatusArgs()...)
	if err != nil {
		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = err.Error()
		}
		return Snapshot{}, fmt.Errorf("cannot read working tree status: %s", msg)
	}
	parsed, err := ParsePorcelain(out)
	if err != nil {
		return Snapshot{}, err
	}
	s.Staged = parsed.Staged
	s.Unstaged = parsed.Unstaged
	s.Untracked = parsed.Untracked
	s.Ignored = parsed.Ignored
	s.Unmerged = parsed.Unmerged
	return s, nil
}

// ParsePorcelain parses git status --porcelain=v1 -z output.
func ParsePorcelain(data string) (Snapshot, error) {
	b := []byte(data)
	var s Snapshot
	i := 0
	for i < len(b) {
		if i+3 > len(b) {
			return Snapshot{}, fmt.Errorf("working tree status is truncated")
		}
		xy := b[i : i+2]
		if b[i+2] != ' ' {
			return Snapshot{}, fmt.Errorf("working tree status is truncated")
		}
		i += 3
		path, next, err := readNUL(b, i)
		if err != nil {
			return Snapshot{}, err
		}
		i = next
		if xy[0] == 'R' || xy[0] == 'C' || xy[1] == 'R' || xy[1] == 'C' {
			path, next, err = readNUL(b, i)
			if err != nil {
				return Snapshot{}, err
			}
			i = next
		}
		code := string(xy)
		switch {
		case code == "??":
			s.Untracked = append(s.Untracked, path)
		case code == "!!":
			s.Ignored = append(s.Ignored, path)
		case unmergedCode(code):
			s.Unmerged = append(s.Unmerged, path)
		default:
			if xy[0] != ' ' && xy[0] != '?' && xy[0] != '!' {
				s.Staged = append(s.Staged, path)
			}
			if xy[1] != ' ' && xy[1] != '?' && xy[1] != '!' {
				s.Unstaged = append(s.Unstaged, path)
			}
		}
	}
	return s, nil
}

func readNUL(b []byte, i int) (string, int, error) {
	j := bytes.IndexByte(b[i:], 0)
	if j < 0 {
		return "", 0, fmt.Errorf("working tree status path is not NUL-terminated")
	}
	return string(b[i : i+j]), i + j + 1, nil
}

func unmergedCode(code string) bool {
	switch code {
	case "DD", "AU", "UD", "UA", "DU", "AA", "UU":
		return true
	default:
		return false
	}
}
