package git

import (
	"errors"
	"strings"
	"testing"
)

func stashEntry(oid, selector, subject string) string {
	return oid + "\x00" + selector + "\x00" + subject
}

func TestDropStash(t *testing.T) {
	const (
		sha1   = "0123456789abcdef0123456789abcdef01234567"
		sha256 = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		list   = "stash list --format=%H%x00%gd%x00%gs"
	)
	quoted := "keep\t\"quoted\""
	for _, tc := range []struct {
		name    string
		oid     string
		message string
		list    string
		want    string
		err     string
	}{
		{name: "only", oid: "owned", message: "keep", list: stashEntry("owned", "stash@{0}", "keep"), want: "stash@{0}"},
		{name: "newest", oid: "owned", message: "keep", list: stashEntry("owned", "stash@{0}", "keep") + "\n" + stashEntry("other", "stash@{1}", "no"), want: "stash@{0}"},
		{name: "middle", oid: "owned", message: "keep", list: stashEntry("new", "stash@{0}", "no") + "\n" + stashEntry("owned", "stash@{1}", "On main: keep") + "\n" + stashEntry("old", "stash@{2}", "no"), want: "stash@{1}"},
		{name: "oldest", oid: "owned", message: "keep", list: stashEntry("new", "stash@{0}", "no") + "\n" + stashEntry("mid", "stash@{1}", "no") + "\n" + stashEntry("owned", "stash@{2}", "keep"), want: "stash@{2}"},
		{name: "missing", oid: "owned", message: "keep", list: stashEntry("other", "stash@{0}", "no")},
		{name: "empty", oid: "owned", message: "keep"},
		{name: "changed", oid: "owned", message: "keep", list: stashEntry("owned", "stash@{0}", "other"), err: "changed identity"},
		{name: "duplicate", oid: "owned", message: "keep", list: stashEntry("owned", "stash@{0}", "keep") + "\n" + stashEntry("owned", "stash@{2}", "On main: keep"), err: "ambiguous"},
		{name: "same oid different message", oid: "owned", message: "keep", list: stashEntry("owned", "stash@{0}", "keep") + "\n" + stashEntry("owned", "stash@{1}", "other"), want: "stash@{0}"},
		{name: "uuid prefix", oid: "owned", message: "eg work.sync auto-stash abc", list: stashEntry("owned", "stash@{0}", "eg work.sync auto-stash abc") + "\n" + stashEntry("other", "stash@{1}", "eg work.sync auto-stash abcdef"), want: "stash@{0}"},
		{name: "tab and quotes", oid: "owned", message: quoted, list: stashEntry("owned", "stash@{0}", quoted), want: "stash@{0}"},
		{name: "sha256", oid: sha256, message: "keep", list: stashEntry(sha256, "stash@{0}", "keep") + "\n" + stashEntry(sha1, "stash@{1}", "keep"), want: "stash@{0}"},
		{name: "sha1", oid: sha1, message: "keep", list: stashEntry(sha256, "stash@{0}", "keep") + "\n" + stashEntry(sha1, "stash@{1}", "On main: keep"), want: "stash@{1}"},
		{name: "oid only", oid: "owned", list: stashEntry("owned", "stash@{1}", "whatever"), want: "stash@{1}"},
		{name: "oid only duplicate", oid: "owned", list: stashEntry("owned", "stash@{0}", "a") + "\n" + stashEntry("owned", "stash@{1}", "b"), err: "ambiguous"},
		{name: "malformed", oid: "owned", message: "keep", list: "owned", err: "cannot be identified"},
		{name: "bad selector", oid: "owned", message: "keep", list: stashEntry("owned", "stash@{0} ", "keep"), err: "cannot be identified"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := NewMemoryRunner()
			Use(m)
			t.Cleanup(func() { Use(RealRunner{}) })
			m.Outputs[list] = tc.list
			err := DropStash(tc.oid, tc.message)
			if tc.err != "" {
				if err == nil || !strings.Contains(err.Error(), tc.err) {
					t.Fatalf("err=%v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			var dropped []string
			for _, call := range m.Calls {
				if len(call.Args) == 3 && call.Args[0] == "stash" && call.Args[1] == "drop" {
					dropped = append(dropped, call.Args[2])
				}
			}
			if tc.want == "" && len(dropped) != 0 || tc.want != "" && (len(dropped) != 1 || dropped[0] != tc.want) {
				t.Fatalf("dropped=%v want=%q", dropped, tc.want)
			}
		})
	}
}

func TestDropStashRequiresCommit(t *testing.T) {
	m := NewMemoryRunner()
	Use(m)
	t.Cleanup(func() { Use(RealRunner{}) })
	if err := DropStash("", "keep"); err == nil || !strings.Contains(err.Error(), "without a commit") {
		t.Fatalf("err=%v", err)
	}
	if len(m.Calls) != 0 {
		t.Fatalf("calls=%v", m.Calls)
	}
}

func TestDropStashKeepsSelectorChosenBeforeDrop(t *testing.T) {
	m := NewMemoryRunner()
	Use(m)
	t.Cleanup(func() {
		BeforeDropStash = nil
		Use(RealRunner{})
	})
	const list = "stash list --format=%H%x00%gd%x00%gs"
	m.Outputs[list] = stashEntry("owned", "stash@{1}", "keep") + "\n" + stashEntry("other", "stash@{0}", "no")
	BeforeDropStash = func() {
		m.Outputs[list] = stashEntry("owned", "stash@{0}", "keep") + "\n" + stashEntry("other", "stash@{1}", "no")
	}
	if err := DropStash("owned", "keep"); err != nil {
		t.Fatal(err)
	}
	var dropped []string
	for _, call := range m.Calls {
		if len(call.Args) == 3 && call.Args[0] == "stash" && call.Args[1] == "drop" {
			dropped = append(dropped, call.Args[2])
		}
	}
	if len(dropped) != 1 || dropped[0] != "stash@{1}" {
		t.Fatalf("dropped=%v", dropped)
	}
}

func TestDropStashPropagatesGitErrors(t *testing.T) {
	const list = "stash list --format=%H%x00%gd%x00%gs"
	for _, command := range []string{list, "stash drop stash@{1}"} {
		t.Run(command, func(t *testing.T) {
			m := NewMemoryRunner()
			Use(m)
			t.Cleanup(func() { Use(RealRunner{}) })
			m.Outputs[list] = stashEntry("owned", "stash@{1}", "keep")
			failure := errors.New("git failed")
			m.FailOn[command] = failure
			if err := DropStash("owned", "keep"); !errors.Is(err, failure) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
