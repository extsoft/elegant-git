package prompt

import "testing"

func TestRequiredLine(t *testing.T) {
	tests := []struct {
		prompt, suggestion, want string
	}{
		{"Workspace name", "", "Workspace name:"},
		{"Git user.name", "Alice", "Git user.name [Alice] (press enter to accept):"},
		{"Git user.email", "alice@example.com", "Git user.email [alice@example.com] (press enter to accept):"},
	}
	for _, tc := range tests {
		if got := RequiredLine(tc.prompt, tc.suggestion); got != tc.want {
			t.Errorf("RequiredLine(%q, %q) = %q, want %q", tc.prompt, tc.suggestion, got, tc.want)
		}
	}
}

func TestOptionalLine(t *testing.T) {
	tests := []struct {
		prompt, suggestion, want string
	}{
		{"Signing key", "", "Signing key (press enter to skip):"},
		{"Signing key", "ABC123", "Signing key [ABC123]:"},
		{"Editor command", "vim", "Editor command [vim]:"},
	}
	for _, tc := range tests {
		if got := OptionalLine(tc.prompt, tc.suggestion); got != tc.want {
			t.Errorf("OptionalLine(%q, %q) = %q, want %q", tc.prompt, tc.suggestion, got, tc.want)
		}
	}
}

func TestClosedLine(t *testing.T) {
	tests := []struct {
		prompt  string
		options []string
		def     string
		want    string
	}{
		{
			"Apply workspace \"work\" to current repository?",
			yesNoOptions,
			"yes",
			"Apply workspace \"work\" to current repository? [yes/no] (press enter to 'yes'):",
		},
		{
			"Override existing workspace \"home\"?",
			yesNoOptions,
			"no",
			"Override existing workspace \"home\"? [yes/no] (press enter to 'no'):",
		},
		{
			"Apply to repo?",
			batchChoiceOptions,
			"no",
			"Apply to repo? [yes/no/all/skip] (press enter to 'no'):",
		},
		{
			"Hook type",
			[]string{"ahead", "after"},
			"",
			"Hook type [ahead/after]:",
		},
	}
	for _, tc := range tests {
		if got := ClosedLine(tc.prompt, tc.options, tc.def); got != tc.want {
			t.Errorf("ClosedLine(...) = %q, want %q", got, tc.want)
		}
	}
}

func TestMatchClosed(t *testing.T) {
	tests := []struct {
		input, def string
		options    []string
		want       string
		ok         bool
	}{
		{"yes", "no", yesNoOptions, "yes", true},
		{"y", "no", yesNoOptions, "yes", true},
		{"Y", "no", yesNoOptions, "yes", true},
		{"no", "yes", yesNoOptions, "no", true},
		{"n", "yes", yesNoOptions, "no", true},
		{"", "yes", yesNoOptions, "yes", true},
		{"", "no", yesNoOptions, "no", true},
		{"", "", yesNoOptions, "", false},
		{"ye", "no", yesNoOptions, "", false},
		{"x", "no", yesNoOptions, "", false},
		{"all", "no", batchChoiceOptions, "all", true},
		{"a", "no", batchChoiceOptions, "all", true},
		{"skip", "no", batchChoiceOptions, "skip", true},
		{"s", "no", batchChoiceOptions, "skip", true},
		{"", "skip", batchChoiceOptions, "skip", true},
		{"a", "", []string{"ahead", "after"}, "", false},
		{"ahead", "", []string{"ahead", "after"}, "ahead", true},
		{"after", "", []string{"ahead", "after"}, "after", true},
	}
	for _, tc := range tests {
		got, ok := MatchClosed(tc.input, tc.options, tc.def)
		if got != tc.want || ok != tc.ok {
			t.Errorf("MatchClosed(%q, %v, %q) = (%q, %v), want (%q, %v)",
				tc.input, tc.options, tc.def, got, ok, tc.want, tc.ok)
		}
	}
}
