// Package config implements Elegant Git configuration helpers.
package config

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/bees-hive/elegant-git/internal/cli/legacy"
	"github.com/bees-hive/elegant-git/internal/deprecation"
	"github.com/bees-hive/elegant-git/internal/git"
	memrepo "github.com/bees-hive/elegant-git/internal/memory/repo"
	"github.com/bees-hive/elegant-git/internal/memory/shared"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/bees-hive/elegant-git/internal/version"
)

const (
	DefaultBranchKey      = "elegant-git.default-branch"
	DefaultBranchDefault  = "main"
	ProtectedBranchesKey  = "elegant-git.protected-branches"
	ProtectedBranchesDef  = "main"
	AcquiredKey           = "elegant-git.acquired"
	AcquiredValueLegacy   = "true"
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
	if s, _, err := memrepo.LoadFromCWD(); err == nil && s.DefaultBranch != "" {
		return s.DefaultBranch
	}
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
	if s, _, err := memrepo.LoadFromCWD(); err == nil {
		return memrepo.FreshestDefaultBranch(s)
	}
	if strings.TrimSpace(git.OutputOK("remote")) == "" {
		return DefaultBranch()
	}
	return DefaultRemoteTrackingBranch()
}

// ProtectedBranches returns space-separated protected branch names.
func ProtectedBranches() string {
	if s, _, err := memrepo.LoadFromCWD(); err == nil && len(s.ProtectedBranches) > 0 {
		return memrepo.ProtectedBranchesString(s)
	}
	v := gitConfigLocal(ProtectedBranchesKey)
	if v == "" {
		return ProtectedBranchesDef
	}
	return v
}

// IsBranchProtected reports whether name is a protected branch.
func IsBranchProtected(name string) bool {
	if s, _, err := memrepo.LoadFromCWD(); err == nil {
		return memrepo.IsBranchProtected(s, name)
	}
	for _, b := range strings.Fields(ProtectedBranches()) {
		if b == name {
			return true
		}
	}
	return false
}

// IsGitAcquired reports whether global Elegant Git configuration is recorded.
func IsGitAcquired() bool {
	s, err := shared.Load()
	if err == nil && shared.Acquired(s) != "" {
		return true
	}
	out, err := git.Output("config", "--global", "--get", AcquiredKey)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// AcquiredVersion returns the recorded global install version from shared memory,
// or from legacy git config when not yet migrated.
func AcquiredVersion() string {
	s, err := shared.Load()
	if err == nil {
		if v := shared.Acquired(s); v != "" {
			return v
		}
	}
	return strings.TrimSpace(git.OutputOK("config", "--global", "--get", AcquiredKey))
}

// NeedsLocalGitInstall reports whether repo configure should apply local
// standards and git aliases (false when global elegant-git is already acquired).
func NeedsLocalGitInstall() bool {
	return !IsGitAcquired()
}

// ConfigField describes one interactive git config key.
type ConfigField struct {
	Key, Message, DefaultVal string
}

// BasicsConfiguration interactively sets user.name, user.email, core.editor.
func BasicsConfiguration(scope string, onlyUnset bool, p prompt.Prompter) error {
	return basicsConfiguration(scope, onlyUnset, p, []ConfigField{
		{"user.name", "Git user.name", configGet(scope, "user.name")},
		{"user.email", "Git user.email", configGet(scope, "user.email")},
		{"core.editor", "Editor command", defaultEditor(scope)},
	})
}

// RepositoryBasicsConfiguration sets user and editor locally (workspace flow handles identity).
func RepositoryBasicsConfiguration(scope string, p prompt.Prompter) error {
	return basicsConfiguration(scope, false, p, []ConfigField{
		{"user.name", "Git user.name", configGet(scope, "user.name")},
		{"user.email", "Git user.email", configGet(scope, "user.email")},
		{"core.editor", "Editor command", defaultEditor(scope)},
	})
}

// MarkAcquired records global configuration in shared memory.
func MarkAcquired(_ string) error {
	s, err := shared.Load()
	if err != nil {
		return err
	}
	shared.SetAcquired(s, version.Version)
	if err := shared.Save(s); err != nil {
		return err
	}
	return RemoveObsoleteAcquired("--global", false)
}

func basicsConfiguration(scope string, onlyUnset bool, p prompt.Prompter, fields []ConfigField) error {
	text.InfoBox("Configuring basics...")
	for _, f := range fields {
		if onlyUnset {
			if cur := configGet(scope, f.Key); cur != "" {
				text.CommandText("git", "config", scope, f.Key, cur)
				continue
			}
		}
		answer, err := ask(p, f.Message, f.DefaultVal, f.Key != "core.editor")
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

// AliasesRemoving removes elegant git aliases from scope.
func AliasesRemoving(scope string, dryRun bool) error {
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
	if dryRun {
		for _, key := range keys {
			fmt.Fprintf(os.Stdout, "  would remove %s\n", key)
		}
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

// AliasesConfiguration sets git aliases mapping legacy names to new elegant paths.
func AliasesConfiguration(scope string) error {
	text.InfoBox("Configuring aliases...")
	for _, legacyName := range legacy.LegacyNames() {
		if legacyName == "show-commands" {
			continue
		}
		val := legacy.AliasValue(legacyName)
		if val == "" {
			continue
		}
		if err := git.Verbose("config", scope, "alias."+legacyName, val); err != nil {
			return err
		}
	}
	return nil
}

// MigrateAcquiredValue imports legacy git config markers into shared memory and unsets them.
func MigrateAcquiredValue(scope string) error {
	if scope == "--global" {
		return importAcquiredToSharedMemory(scope)
	}
	return RemoveObsoleteAcquired(scope, false)
}

func importAcquiredToSharedMemory(scope string) error {
	out, err := git.Output("config", scope, "--get", AcquiredKey)
	if err != nil || strings.TrimSpace(out) == "" {
		return RemoveObsoleteAcquired(scope, false)
	}
	val := strings.TrimSpace(out)
	if val == AcquiredValueLegacy {
		deprecation.Record(deprecation.DEP009, "config key: "+AcquiredKey+"="+AcquiredValueLegacy, "shared memory acquired_version", migrateHint(scope))
		val = version.Version
	}
	s, err := shared.Load()
	if err != nil {
		return err
	}
	if shared.Acquired(s) == "" {
		shared.SetAcquired(s, val)
		if err := shared.Save(s); err != nil {
			return err
		}
	}
	return RemoveObsoleteAcquired(scope, false)
}

func migrateHint(scope string) string {
	if scope == "--global" {
		return "git elegant git migrate"
	}
	return "git elegant repo migrate"
}

// RemoveObsoleteAcquired unsets elegant-git.acquired for scope when present.
func RemoveObsoleteAcquired(scope string, dryRun bool) error {
	out, err := git.Output("config", scope, "--get-regexp", AcquiredKey)
	if err != nil || strings.TrimSpace(out) == "" {
		return nil
	}
	if dryRun {
		fmt.Fprintf(os.Stdout, "  would unset %s\n", AcquiredKey)
		return nil
	}
	text.InfoText("Removing old Elegant Git configuration keys...")
	return git.Verbose("config", scope, "--unset", AcquiredKey)
}

// CleanupRedundantLocalInstall removes local acquired marker and elegant aliases
// when global configuration is already applied.
func CleanupRedundantLocalInstall(dryRun bool) error {
	if err := RemoveObsoleteAcquired("--local", dryRun); err != nil {
		return err
	}
	return AliasesRemoving("--local", dryRun)
}

// ObsoleteConfigurationsRemoving drops legacy acquired marker and elegant aliases.
func ObsoleteConfigurationsRemoving(scope string) error {
	text.InfoBox("Removing obsolete configurations...")
	if scope == "--global" {
		if err := importAcquiredToSharedMemory(scope); err != nil {
			return err
		}
	} else if err := RemoveObsoleteAcquired(scope, false); err != nil {
		return err
	}
	return AliasesRemoving(scope, false)
}

func ask(p prompt.Prompter, message, defaultVal string, required bool) (string, error) {
	if prompt.NonInteractive(p) {
		return defaultVal, nil
	}
	if required {
		return p.EditOrAccept(message, defaultVal)
	}
	return p.Optional(message, defaultVal)
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
