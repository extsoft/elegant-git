package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/legacy"
)

// GitInstall diagnoses global Git installation drift: alias.elegant or an existing
// flat alias that is wrong (a never-configured machine is not a finding), leftover
// git-elegant on PATH, and stale git-elegant completion scripts.
func GitInstall() []Finding {
	var out []Finding
	out = append(out, diagnoseGlobalAliases()...)
	out = append(out, diagnoseStaleCompletion()...)
	out = append(out, diagnoseLegacyBinary()...)
	return out
}

func diagnoseGlobalAliases() []Finding {
	if !globalAliasesDrifted() {
		return nil
	}
	return []Finding{{
		Problem: "global Elegant Git aliases have drifted",
		Repair:  "write alias.elegant = \"!eg\" and rewrite stale flat aliases",
		Apply:   RepairGlobalAliases,
	}}
}

func globalAliasesDrifted() bool {
	elegant := git.ConfigGlobalGet(config.ElegantAliasKey)
	if elegant != "" && elegant != config.ElegantAliasValue {
		return true
	}
	for _, name := range legacy.LegacyNames() {
		if name == "show-commands" {
			continue
		}
		cur := git.ConfigGlobalGet("alias." + name)
		if cur == "" {
			continue
		}
		want := legacy.AliasValue(name)
		if want != "" && cur != want {
			return true
		}
	}
	return false
}

// RepairGlobalAliases writes alias.elegant and rewrites existing drifted flat aliases.
func RepairGlobalAliases() error {
	if err := config.EnsureElegantAlias("--global", false); err != nil {
		return err
	}
	for _, name := range legacy.LegacyNames() {
		if name == "show-commands" {
			continue
		}
		key := "alias." + name
		cur := git.ConfigGlobalGet(key)
		if cur == "" {
			continue
		}
		newVal := legacy.AliasValue(name)
		if newVal == "" || cur == newVal {
			continue
		}
		if err := git.ConfigSet("--global", key, newVal); err != nil {
			return err
		}
	}
	return nil
}

func diagnoseStaleCompletion() []Finding {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	candidates := []string{
		filepath.Join(home, ".local/share/bash-completion/completions/git-elegant"),
		filepath.Join(home, ".local/share/bash-completion/completions/_git-elegant"),
		filepath.Join(home, ".zsh/completions/_git-elegant"),
		filepath.Join(home, ".config/zsh/completions/_git-elegant"),
	}
	var found []string
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			found = append(found, p)
		}
	}
	if len(found) == 0 {
		return nil
	}
	return []Finding{{
		Problem: "stale git-elegant completion script(s): " + strings.Join(found, ", "),
		Repair:  "delete them and run `eg completion <shell>`",
	}}
}

func diagnoseLegacyBinary() []Finding {
	path, err := exec.LookPath("git-elegant")
	if err != nil {
		return nil
	}
	return []Finding{{
		Problem: fmt.Sprintf("git-elegant is still on PATH (%s)", path),
		Repair:  "remove it so `git elegant` uses alias.elegant = \"!eg\"",
	}}
}
