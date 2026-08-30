package git

import (
	"fmt"
	"strings"
	"sync"

	"github.com/extsoft/elegant-git/internal/text"
)

// Call records one git invocation.
type Call struct {
	Args []string
	Op   bool
}

// MemoryRepo holds minimal in-memory repository state for tests.
type MemoryRepo struct {
	Branches      []string
	CurrentBranch string
	Remotes       []string
	ModifiedFiles []string
	UnstagedFiles []string
	LocalConfig   map[string]string
}

// MemoryRunner simulates git for unit tests.
type MemoryRunner struct {
	mu sync.Mutex

	GlobalConfig map[string]string
	Repo         *MemoryRepo
	Calls        []Call
	Outputs      map[string]string
	FailOn       map[string]error
}

// NewMemoryRunner returns a runner with empty global config and a default repo.
func NewMemoryRunner() *MemoryRunner {
	return &MemoryRunner{
		GlobalConfig: map[string]string{},
		Repo: &MemoryRepo{
			Branches:      []string{"main"},
			CurrentBranch: "main",
			LocalConfig:   map[string]string{},
		},
		Outputs: map[string]string{},
		FailOn:  map[string]error{},
	}
}

func (m *MemoryRunner) key(args []string) string {
	return strings.Join(args, " ")
}

func (m *MemoryRunner) record(args []string, op bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, Call{Args: append([]string(nil), args...), Op: op})
}

func (m *MemoryRunner) canned(args []string) (string, bool) {
	if m.Outputs == nil {
		return "", false
	}
	if out, ok := m.Outputs[m.key(args)]; ok {
		return out, true
	}
	return "", false
}

func (m *MemoryRunner) fail(args []string) error {
	if m.FailOn == nil {
		return nil
	}
	return m.FailOn[m.key(args)]
}

func (m *MemoryRunner) scopeIndex(args []string) int {
	for i, a := range args {
		if a == "--global" || a == "--local" {
			return i
		}
	}
	return -1
}

func (m *MemoryRunner) configMap(scope string) map[string]string {
	if scope == "--global" {
		if m.GlobalConfig == nil {
			m.GlobalConfig = map[string]string{}
		}
		return m.GlobalConfig
	}
	if m.Repo == nil {
		m.Repo = &MemoryRepo{LocalConfig: map[string]string{}}
	}
	if m.Repo.LocalConfig == nil {
		m.Repo.LocalConfig = map[string]string{}
	}
	return m.Repo.LocalConfig
}

func (m *MemoryRunner) handleConfig(args []string) (string, error) {
	scope := "--local"
	if i := m.scopeIndex(args); i >= 0 {
		scope = args[i]
		args = append(append([]string(nil), args[:i]...), args[i+1:]...)
	}
	if len(args) < 2 {
		return "", nil
	}
	cfg := m.configMap(scope)
	switch args[0] {
	case "config":
		if len(args) >= 3 && args[1] == "--get-regexp" {
			var lines []string
			prefix := strings.ReplaceAll(strings.TrimPrefix(args[2], "^"), `\.`, ".")
			for k, v := range cfg {
				if strings.HasPrefix(k, prefix) {
					lines = append(lines, k+" "+v)
				}
			}
			return strings.Join(lines, "\n"), nil
		}
		if len(args) >= 3 && args[1] == "--get" {
			if v, ok := cfg[args[2]]; ok {
				return v, nil
			}
			return "", fmt.Errorf("not found")
		}
		if len(args) >= 3 && args[1] == "--unset" {
			delete(cfg, args[2])
			return "", nil
		}
		if len(args) >= 3 {
			cfg[args[1]] = strings.Join(args[2:], " ")
			return "", nil
		}
	}
	return "", nil
}

func (m *MemoryRunner) simulate(args []string) (string, error) {
	if err := m.fail(args); err != nil {
		return "", err
	}
	if out, ok := m.canned(args); ok {
		return out, nil
	}
	if len(args) > 0 && args[0] == "config" {
		return m.handleConfig(args)
	}
	switch m.key(args) {
	case "rev-parse --show-toplevel":
		return "/repo", nil
	case "rev-parse --abbrev-ref HEAD":
		if m.Repo != nil && m.Repo.CurrentBranch != "" {
			return m.Repo.CurrentBranch, nil
		}
		return "main", nil
	case "remote":
		if m.Repo != nil && len(m.Repo.Remotes) > 0 {
			return strings.Join(m.Repo.Remotes, "\n"), nil
		}
		return "", nil
	case "rev-parse --git-path rebase-merge", "rev-parse --git-path rebase-apply":
		return "", fmt.Errorf("not found")
	}
	return "", nil
}

func (m *MemoryRunner) Verbose(args ...string) error {
	m.record(args, false)
	text.CommandText(append([]string{"git"}, args...)...)
	_, err := m.simulate(args)
	return err
}

func (m *MemoryRunner) VerboseOp(processor func(string), args ...string) error {
	m.record(args, true)
	text.CommandText(append([]string{"git"}, args...)...)
	out, err := m.simulate(args)
	if err != nil {
		return err
	}
	processor(out)
	return nil
}

func (m *MemoryRunner) VerboseOpLines(lineFn func(string), args ...string) error {
	m.record(args, true)
	text.CommandText(append([]string{"git"}, args...)...)
	return m.emitLines(lineFn, args)
}

func (m *MemoryRunner) StreamLines(lineFn func(string), args ...string) error {
	m.record(args, true)
	return m.emitLines(lineFn, args)
}

func (m *MemoryRunner) emitLines(lineFn func(string), args []string) error {
	out, err := m.simulate(args)
	if out == "" {
		if canned, ok := m.canned(args); ok {
			out = canned
		}
	}
	for _, line := range strings.Split(out, "\n") {
		if line != "" && lineFn != nil {
			lineFn(line)
		}
	}
	return err
}

func (m *MemoryRunner) Output(args ...string) (string, error) {
	m.record(args, false)
	return m.simulate(args)
}

func (m *MemoryRunner) OutputOK(args ...string) string {
	out, _ := m.Output(args...)
	return strings.TrimSpace(out)
}
