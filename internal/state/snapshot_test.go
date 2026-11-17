package state

import "testing"

func TestParsePorcelainSeesUntrackedAndNUL(t *testing.T) {
	in := " M tracked\x00A  added\x00MM both\x00?? odd name\x00?? has\nline\x00!! ignored\x00UU conflict\x00R  old\x00renamed\x00"
	got, err := ParsePorcelain(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Untracked) != 2 || got.Untracked[0] != "odd name" || got.Untracked[1] != "has\nline" {
		t.Fatalf("untracked %q", got.Untracked)
	}
	if len(got.Ignored) != 1 || got.Ignored[0] != "ignored" {
		t.Fatalf("ignored %q", got.Ignored)
	}
	if len(got.Unmerged) != 1 || got.Unmerged[0] != "conflict" {
		t.Fatalf("unmerged %q", got.Unmerged)
	}
	if len(got.Staged) != 3 || got.Staged[0] != "added" || got.Staged[1] != "both" || got.Staged[2] != "renamed" {
		t.Fatalf("staged %q", got.Staged)
	}
	if len(got.Unstaged) != 2 || got.Unstaged[0] != "tracked" || got.Unstaged[1] != "both" {
		t.Fatalf("unstaged %q", got.Unstaged)
	}
	if !got.Relevant() || !got.HasTracked() {
		t.Fatal("expected tracked dirt")
	}
}

func TestParsePorcelainEmptyIsClean(t *testing.T) {
	got, err := ParsePorcelain("")
	if err != nil {
		t.Fatal(err)
	}
	if got.Relevant() || got.HasTracked() {
		t.Fatalf("%+v", got)
	}
}

func TestParsePorcelainRejectsTruncation(t *testing.T) {
	if _, err := ParsePorcelain("?? missing"); err == nil {
		t.Fatal("expected error")
	}
}
