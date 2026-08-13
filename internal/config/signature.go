package config

import (
	"os"
	"os/exec"
	"strings"

	"github.com/bees-hive/elegant-git/internal/git"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/bees-hive/elegant-git/internal/text"
)

// ConfigureSignature optionally sets GPG signing for the local repository.
func ConfigureSignature(p prompt.Prompter) error {
	if _, err := exec.LookPath("gpg"); err != nil {
		return nil
	}
	text.InfoBox("Configuring signature...")
	email, _ := git.Output("config", "--local", "user.email")
	email = strings.TrimSpace(email)
	listkeys := "gpg --list-secret-keys --keyid-format long " + email
	text.CommandText(listkeys)
	cmd := exec.Command("gpg", "--list-secret-keys", "--keyid-format", "long", email)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		text.InfoText("There is no gpg key for the given email.")
		text.InfoText("A signature is not configured.")
		return nil
	}
	text.InfoText("From the list of GPG keys above, copy the GPG key ID you'd like to use.")
	text.InfoText("It will be")
	text.InfoText("    3AA5C34371567BD2")
	text.InfoText("for the output like this")
	text.InfoText("    sec   4096R/3AA5C34371567BD2 2016-03-10 [expires: 2017-03-10]")
	text.InfoText("    A330C91F8EC4BC7AECFA63E03AA5C34371567BD2")
	text.InfoText("    uid                          Hubot")
	text.InfoText("")
	if prompt.NonInteractive(p) {
		return nil
	}
	key, err := p.Optional("Signing key", "")
	if err != nil {
		return err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		text.InfoText("The signature is not configured as the empty key is provided.")
		return nil
	}
	gpgPath, _ := exec.LookPath("gpg")
	if err := git.Verbose("config", "--local", "user.signingkey", key); err != nil {
		return err
	}
	if err := git.Verbose("config", "--local", "gpg.program", gpgPath); err != nil {
		return err
	}
	if err := git.Verbose("config", "--local", "commit.gpgsign", "true"); err != nil {
		return err
	}
	if err := git.Verbose("config", "--local", "tag.forceSignAnnotated", "true"); err != nil {
		return err
	}
	return git.Verbose("config", "--local", "tag.gpgSign", "true")
}
