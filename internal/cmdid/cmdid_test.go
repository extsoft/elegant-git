package cmdid

import "testing"

func TestHookFileName(t *testing.T) {
	id := ID{Command: "work", Action: "save"}
	if got := id.HookFileName("ahead"); got != "work-save-ahead" {
		t.Fatalf("got %q want work-save-ahead", got)
	}
}
