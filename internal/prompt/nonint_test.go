package prompt

import "testing"

func TestNonInteractiveStringFails(t *testing.T) {
	p := NewNonInteractive()
	if _, err := p.String("q", ""); err != ErrNonInteractive {
		t.Fatalf("got %v", err)
	}
}

func TestNonInteractiveConfirmFalse(t *testing.T) {
	p := NewNonInteractive()
	ok, err := p.Confirm("apply?")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false")
	}
}
