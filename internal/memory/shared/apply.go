package shared

import (
	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/prompt"
)

// Apply controls how ApplyProfile writes keys during propagation.
type Apply struct {
	Force bool
	Skip  bool
}

// ApplyProfile writes profile identity into local git config.
func ApplyProfile(p *Profile, prompter prompt.Prompter, a *Apply) error {
	if p == nil {
		return nil
	}
	if a == nil {
		a = &Apply{}
	}
	if a.Skip {
		return nil
	}
	if err := applyKey("user.name", p.UserName, true, prompter, a); err != nil {
		return err
	}
	if err := applyKey("user.email", p.UserEmail, true, prompter, a); err != nil {
		return err
	}
	if err := applyOptionalKey("user.signingkey", p.SigningKey, "Signing key", prompter, a); err != nil {
		return err
	}
	if err := applyOptionalKey("gpg.program", p.GPGProgram, "GPG program", prompter, a); err != nil {
		return err
	}
	return applyOptionalKey("core.editor", p.Editor, "Editor command", prompter, a)
}

func applyKey(key, value string, required bool, prompter prompt.Prompter, a *Apply) error {
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
	v, err := prompter.EditOrAccept(key, value)
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
