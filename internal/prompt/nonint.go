package prompt

// nonInteractive rejects all interactive prompts except Confirm (returns false).
type nonInteractive struct{}

// NewNonInteractive returns a prompter that fails on String/Choose/Required.
func NewNonInteractive() Prompter {
	return &nonInteractive{}
}

func (nonInteractive) String(string, string) (string, error) {
	return "", ErrNonInteractive
}

func (nonInteractive) Confirm(string) (bool, error) {
	return false, nil
}

func (nonInteractive) Choose(string, []string) (int, error) {
	return -1, ErrNonInteractive
}

func (nonInteractive) Required(string, string) error {
	return ErrNonInteractive
}

func (nonInteractive) EditOrAccept(string, suggested string) (string, error) {
	return suggested, nil
}

func (nonInteractive) BatchChoice(string) (BatchDecision, error) {
	return BatchSkip, nil
}
