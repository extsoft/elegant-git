// Package config implements Elegant Git configuration helpers.
package config

import (
	"bufio"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/text"
)

const (
	DefaultBranchKey      = "elegant-git.default-branch"
	DefaultBranchDefault  = "master"
	ProtectedBranchesKey  = "elegant-git.protected-branches"
	ProtectedBranchesDef  = "master"
	AcquiredKey           = "elegant-git.acquired"
	AcquiredValue         = "true"
	DefaultUpstreamRemote = "origin"
)

type stdPair struct {
	key    string
	value  string
	osSkip func(goos string) bool
}

var standardPairs = []stdPair{
	{key: "core.commentChar", value: "|"},
	{key: "apply.whitespace", value: "fix"},
	{key: "fetch.prune", value: "true"},
	{key: "fetch.pruneTags", value: "false"},
	{key: "core.autocrlf", value: "input", osSkip: skipUnlessDarwinLinux},
	{key: "core.autocrlf", value: "true", osSkip: skipUnlessWindows},
	{key: "pull.rebase", value: "true"},
	{key: "rebase.autoStash", value: "false"},
	{key: "credential.helper", value: "osxkeychain", osSkip: skipUnlessDarwin},
}

// DefaultBranch returns the configured default branch name.
func DefaultBranch() string {
	v := gitConfigLocal(DefaultBranchKey)
	if v == "" {
		return DefaultBranchDefault
	}
	return v
}

// DefaultRemoteTrackingBranch returns origin/<default-branch>.
func DefaultRemoteTrackingBranch() string {
	return DefaultUpstreamRemote + "/" + DefaultBranch()
}

// FreshestDefaultBranch returns the remote tracking branch if remotes exist.
func FreshestDefaultBranch() string {
	if strings.TrimSpace(git.OutputOK("remote")) == "" {
		return DefaultBranch()
	}
	return DefaultRemoteTrackingBranch()
}

// ProtectedBranches returns space-separated protected branch names.
func ProtectedBranches() string {
	v := gitConfigLocal(ProtectedBranchesKey)
	if v == "" {
		return ProtectedBranchesDef
	}
	return v
}

// IsBranchProtected reports whether name is a protected branch.
func IsBranchProtected(name string) bool {
	for _, b := range strings.Fields(ProtectedBranches()) {
		if b == name {
			return true
		}
	}
	return false
}

// IsGitAcquired reports whether global elegant-git.acquired is true.
func IsGitAcquired() bool {
	out, err := git.Output("config", "--global", "--get", AcquiredKey)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == AcquiredValue
}

// ConfigField describes one interactive git config key.
type ConfigField struct {
	Key, Message, DefaultVal string
}

// BasicsConfiguration interactively sets user.name, user.email, core.editor.
func BasicsConfiguration(scope string, onlyUnset bool, reader io.Reader) error {
	return basicsConfiguration(scope, onlyUnset, reader, []ConfigField{
		{"user.name", "What is your user name?", configGet(scope, "user.name")},
		{"user.email", "What is your user email?", configGet(scope, "user.email")},
		{"core.editor", "What is the command to launching an editor?", defaultEditor(scope)},
	})
}

// RepositoryBasicsConfiguration sets user, editor, default branch, and protected branches locally.
func RepositoryBasicsConfiguration(scope string, reader io.Reader) error {
	return basicsConfiguration(scope, false, reader, []ConfigField{
		{"user.name", "What is your user name?", configGet(scope, "user.name")},
		{"user.email", "What is your user email?", configGet(scope, "user.email")},
		{"core.editor", "What is the command to launching an editor?", defaultEditor(scope)},
		{DefaultBranchKey, "What is the default branch?", defaultBranchDefault(scope)},
		{ProtectedBranchesKey, "What are protected branches (split with space)?", protectedBranchesDefault(scope)},
	})
}

// MarkAcquired sets elegant-git.acquired for scope.
func MarkAcquired(scope string) error {
	return git.Verbose("config", scope, AcquiredKey, AcquiredValue)
}

func basicsConfiguration(scope string, onlyUnset bool, reader io.Reader, fields []ConfigField) error {
	text.InfoBox("Configuring basics...")
	notify := true
	for _, f := range fields {
		if onlyUnset {
			if cur := configGet(scope, f.Key); cur != "" {
				text.CommandText("git", "config", scope, f.Key, cur)
				continue
			}
		}
		if notify {
			text.InfoText("Please hit enter if you wish {default value}.")
			notify = false
		}
		answer, err := ask(reader, f.Message, f.DefaultVal)
		if err != nil {
			return err
		}
		if answer != "" {
			if err := git.Verbose("config", scope, f.Key, answer); err != nil {
				return err
			}
		}
	}
	return nil
}

// StandardsConfiguration applies mandatory git config standards for scope.
func StandardsConfiguration(scope string) error {
	text.InfoBox("Configuring standards...")
	goos := runtime.GOOS
	for _, p := range standardPairs {
		if p.osSkip != nil && p.osSkip(goos) {
			continue
		}
		if err := git.Verbose("config", scope, p.key, p.value); err != nil {
			return err
		}
	}
	return nil
}

// AliasesRemoving removes old elegant git aliases from scope.
func AliasesRemoving(scope string) error {
	out, err := git.Output("config", scope, "--get-regexp", `^alias\.`)
	if err != nil || strings.TrimSpace(out) == "" {
		return nil
	}
	var keys []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := parts[0]
		value := strings.Join(parts[1:], " ")
		if strings.HasPrefix(value, "elegant ") {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	text.InfoText("Removing old Elegant Git aliases...")
	for _, key := range keys {
		if err := git.Verbose("config", scope, "--unset", key); err != nil {
			return err
		}
	}
	text.InfoText(fmt.Sprintf("%d Elegant Git aliases were removed.", len(keys)))
	return nil
}

// AliasesConfiguration sets git aliases for elegant commands.
func AliasesConfiguration(scope string, commands ...string) error {
	text.InfoBox("Configuring aliases...")
	for _, cmd := range commands {
		if err := git.Verbose("config", scope, "alias."+cmd, "elegant "+cmd); err != nil {
			return err
		}
	}
	return nil
}

// ObsoleteConfigurationsRemoving drops legacy keys and aliases.
func ObsoleteConfigurationsRemoving(scope string) error {
	text.InfoBox("Removing obsolete configurations...")
	if out, err := git.Output("config", scope, "--get-regexp", AcquiredKey); err == nil && strings.TrimSpace(out) != "" {
		text.InfoText("Removing old Elegnat Git configuration keys...")
		if err := git.Verbose("config", scope, "--unset", AcquiredKey); err != nil {
			return err
		}
	}
	return AliasesRemoving(scope)
}

func ask(reader io.Reader, message, defaultVal string) (string, error) {
	prompt := message
	if defaultVal != "" {
		prompt = message + " {" + defaultVal + "}"
	}
	text.QuestionText(prompt + ": ")
	line, err := bufio.NewReader(reader).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal, nil
	}
	return line, nil
}

func configGet(scope, key string) string {
	out, err := git.Output("config", scope, "--get", key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func gitConfigLocal(key string) string {
	out, err := git.Output("config", key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func protectedBranchesDefault(scope string) string {
	if v := configGet(scope, ProtectedBranchesKey); v != "" {
		return v
	}
	return ProtectedBranchesDef
}

func defaultBranchDefault(scope string) string {
	if v := configGet(scope, DefaultBranchKey); v != "" {
		return v
	}
	return DefaultBranchDefault
}

func defaultEditor(scope string) string {
	if v := configGet(scope, "core.editor"); v != "" {
		return v
	}
	return "vim"
}

func skipUnlessDarwinLinux(goos string) bool { return goos == "windows" }
func skipUnlessWindows(goos string) bool     { return goos != "windows" }
func skipUnlessDarwin(goos string) bool      { return goos != "darwin" }
