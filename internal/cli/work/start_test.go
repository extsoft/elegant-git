package work

import "testing"

func TestParseStartChangesChoice(t *testing.T) {
	tests := []struct {
		in   string
		mode string
		ok   bool
	}{
		{"", "stash", true},
		{"a", "stash", true},
		{"add", "stash", true},
		{"R", "reset", true},
		{"discard", "reset", true},
		{"c", "cancel", true},
		{"quit", "cancel", true},
		{"x", "", false},
	}
	for _, tc := range tests {
		mode, ok := parseStartChangesChoice(tc.in)
		if mode != tc.mode || ok != tc.ok {
			t.Errorf("parseStartChangesChoice(%q) = %q, %v; want %q, %v", tc.in, mode, ok, tc.mode, tc.ok)
		}
	}
}
