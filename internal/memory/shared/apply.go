package shared

import (
	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/prompt"
)

// Apply controls how ApplyWorkspace writes keys during propagation.
type Apply struct {
	Force bool
	Skip  bool
}

// ApplyWorkspace writes workspace identity into local git config.
func ApplyWorkspace(ws *Workspace, prompter prompt.Prompter, a *Apply) error {
	if ws == nil {
		return nil
	}
	if a == nil {
		a = &Apply{}
	}
	if a.Skip {
		return nil
	}
	if err := applyKey("user.name", "Git user.name", ws.UserName, true, prompter, a); err != nil {
		return err
	}
	if err := applyKey("user.email", "Git user.email", ws.UserEmail, true, prompter, a); err != nil {
		return err
	}
	if err := applyOptionalKey("user.signingkey", ws.SigningKey, "Signing key", prompter, a); err != nil {
		return err
	}
	if err := applyOptionalKey("gpg.program", ws.GPGProgram, "GPG program", prompter, a); err != nil {
		return err
	}
	return applyOptionalKey("core.editor", ws.Editor, "Editor command", prompter, a)
}

func applyKey(key, label, value string, required bool, prompter prompt.Prompter, a *Apply) error {
	current := gitLocal(key)
	if a.Skip {
		return nil
	}
	if value == current {
		return nil
	}
	if value == "" {
		if required && current == "" {
			return git.ConfigLocalSet(key, value)
		}
		return nil
	}
	if a.Force {
		return git.ConfigLocalSet(key, value)
	}
	v, err := prompter.EditOrAccept(label, value)
	if err != nil {
		return err
	}
	if v == "" {
		if required {
			return git.ConfigLocalSet(key, value)
		}
		return nil
	}
	if v == current {
		return nil
	}
	return git.ConfigLocalSet(key, v)
}

func applyOptionalKey(key, value, label string, prompter prompt.Prompter, a *Apply) error {
	if value == "" {
		return nil
	}
	current := gitLocal(key)
	if a.Skip {
		return nil
	}
	if value == current {
		return nil
	}
	if a.Force {
		return git.ConfigLocalSet(key, value)
	}
	v, err := prompt.Skippable(prompter, label, current, value)
	if err != nil {
		return err
	}
	if v == "" || v == current {
		return nil
	}
	return git.ConfigLocalSet(key, v)
}

func gitLocal(key string) string {
	return git.ConfigLocalGet(key)
}
