package work

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/pipe"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/recovery"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/spf13/cobra"
)

func TestSnapshotSeesUntrackedWithoutMutation(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.git("checkout", "-q", "main")
	r.git("config", "status.showUntrackedFiles", "no")
	r.write("odd name", "x\n")
	r.enter()
	index := filepath.Join(r.dir, ".git", "index")
	before, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := state.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("status read changed the index")
	}
	if !snap.Relevant() || len(snap.Untracked) != 1 || snap.Untracked[0] != "odd name" {
		t.Fatalf("%+v", snap)
	}
	if err := runSwitchOpt(prompt.NewNonInteractive(), "carry", "", "feat"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/feat" || r.read("odd name") != "x\n" {
		t.Fatalf("head %s file %q", r.headRef(), r.read("odd name"))
	}
}

func TestCarryKeepsIndexSplit(t *testing.T) {
	r := dirtyPair(t)
	r.write("f", "staged\n")
	r.git("add", "f")
	r.write("f", "unstaged\n")
	r.write("added", "new\n")
	r.git("add", "added")
	r.write("free", "u\n")
	r.write(".gitignore", "skip\n")
	r.git("add", ".gitignore")
	r.write("skip", "ignored\n")
	r.enter()
	cached := r.git("diff", "--cached")
	plain := r.git("diff")
	if err := runSwitchOpt(prompt.NewNonInteractive(), "carry", "", "feat"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/feat" {
		t.Fatalf("head %s", r.headRef())
	}
	if r.git("diff", "--cached") != cached || r.git("diff") != plain {
		t.Fatalf("cached %q plain %q", r.git("diff", "--cached"), r.git("diff"))
	}
	if r.read("free") != "u\n" || r.read("skip") != "ignored\n" {
		t.Fatal("untracked or ignored content changed")
	}
	if strings.Contains(r.git("ls-tree", "-r", "--name-only", "stash@{0}"), "skip") {
		t.Fatal("ignored file was stashed")
	}
	if third := r.gitAllow("rev-parse", "--verify", "--quiet", "stash@{0}^3"); third == nil {
		if strings.Contains(r.git("ls-tree", "-r", "--name-only", "stash@{0}^3"), "skip") {
			t.Fatal("ignored file was stashed")
		}
	}
}

func TestDiscardScope(t *testing.T) {
	t.Run("tracked and untracked", func(t *testing.T) {
		r := dirtyPair(t)
		r.write("free", "u\n")
		r.write(".gitignore", "skip\n")
		r.git("add", ".gitignore")
		r.git("commit", "-q", "-m", "ignore")
		r.write("skip", "ignored\n")
		r.enter()
		if err := runSwitchOpt(prompt.NewNonInteractive(), "discard", "", "feat"); err != nil {
			t.Fatal(err)
		}
		if r.headRef() != "refs/heads/feat" || r.read("f") != "base\n" || r.read("free") != "u\n" || r.read("skip") != "ignored\n" {
			t.Fatalf("head %s f %q free %q skip %q", r.headRef(), r.read("f"), r.read("free"), r.read("skip"))
		}
	})
	t.Run("untracked only", func(t *testing.T) {
		r := newSwRepo(t)
		r.git("checkout", "-q", "-b", "feat")
		r.git("checkout", "-q", "main")
		r.write("free", "u\n")
		r.enter()
		buf := captureText(t)
		err := runSwitchOpt(prompt.NewNonInteractive(), "discard", "", "feat")
		assertWorkflow(t, err)
		if !strings.Contains(buf.String(), "carry") || !strings.Contains(buf.String(), "stash") || !strings.Contains(buf.String(), "cancel") {
			t.Fatalf("out=%q", buf.String())
		}
		if r.headRef() != "refs/heads/main" || r.read("free") != "u\n" || r.stashList() != "" {
			t.Fatal("discarded untracked work")
		}
	})
	t.Run("collision", func(t *testing.T) {
		r := newSwRepo(t)
		r.git("checkout", "-q", "-b", "feat")
		r.write("free", "dest\n")
		r.git("add", "free")
		r.git("commit", "-q", "-m", "free")
		r.git("checkout", "-q", "main")
		r.write("f", "dirty\n")
		r.write("free", "local\n")
		r.enter()
		err := runSwitchOpt(prompt.NewNonInteractive(), "discard", "", "feat")
		assertWorkflow(t, err)
		if r.headRef() != "refs/heads/main" || r.read("f") != "dirty\n" || r.read("free") != "local\n" {
			t.Fatalf("head %s f %q free %q", r.headRef(), r.read("f"), r.read("free"))
		}
	})
}

func TestIgnoredAndNestedAreRefused(t *testing.T) {
	t.Run("ignored", func(t *testing.T) {
		r := newSwRepo(t)
		r.git("checkout", "-q", "-b", "feat")
		r.write("secret", "dest\n")
		r.git("add", "secret")
		r.git("commit", "-q", "-m", "secret")
		r.git("checkout", "-q", "main")
		r.write(".gitignore", "secret\n")
		r.git("add", ".gitignore")
		r.git("commit", "-q", "-m", "ignore")
		r.write("secret", "local\n")
		r.write("f", "dirty\n")
		r.enter()
		err := runSwitchOpt(prompt.NewNonInteractive(), "carry", "", "feat")
		assertWorkflow(t, err)
		if r.headRef() != "refs/heads/main" || r.read("secret") != "local\n" || r.read("f") != "dirty\n" || r.stashList() != "" {
			t.Fatalf("head %s secret %q stash %q", r.headRef(), r.read("secret"), r.stashList())
		}
	})
	t.Run("nested", func(t *testing.T) {
		r := dirtyPair(t)
		nested := filepath.Join(r.dir, "nested")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatal(err)
		}
		gitDir(t, nested, "init", "-q", "-b", "main")
		if err := os.WriteFile(filepath.Join(nested, "inside"), []byte("keep\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		r.enter()
		buf := captureText(t)
		err := runSwitchOpt(prompt.NewNonInteractive(), "stash", "", "feat")
		assertWorkflow(t, err)
		if !strings.Contains(buf.String(), "nested") {
			t.Fatalf("out=%q", buf.String())
		}
		if r.headRef() != "refs/heads/main" || r.read("f") != "dirty\n" || r.stashList() != "" {
			t.Fatal("nested repo was moved")
		}
		if got := string(mustRead(t, filepath.Join(nested, "inside"))); got != "keep\n" {
			t.Fatal(got)
		}
	})
	t.Run("symlink", func(t *testing.T) {
		r := newSwRepo(t)
		r.git("checkout", "-q", "-b", "feat")
		r.write("link", "dest\n")
		r.git("add", "link")
		r.git("commit", "-q", "-m", "link")
		r.git("checkout", "-q", "main")
		r.write(".gitignore", "link\n")
		r.git("add", ".gitignore")
		r.git("commit", "-q", "-m", "ignore")
		if err := os.Symlink("somewhere", filepath.Join(r.dir, "link")); err != nil {
			t.Fatal(err)
		}
		r.write("f", "dirty\n")
		r.enter()
		err := runSwitchOpt(prompt.NewNonInteractive(), "carry", "", "feat")
		assertWorkflow(t, err)
		info, statErr := os.Lstat(filepath.Join(r.dir, "link"))
		if statErr != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("link %v %v", info, statErr)
		}
		if r.stashList() != "" || r.read("f") != "dirty\n" {
			t.Fatal("mutated")
		}
	})
}

func TestStashOnlyHandoff(t *testing.T) {
	r := dirtyPair(t)
	r.write("f", "staged\n")
	r.git("add", "f")
	r.write("f", "unstaged\n")
	r.write("free", "u\n")
	r.enter()
	buf := captureText(t)
	if err := runSwitchOpt(prompt.NewNonInteractive(), "stash", "same label", "feat"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/feat" || r.read("f") != "base\n" {
		t.Fatalf("head %s f %q", r.headRef(), r.read("f"))
	}
	if _, err := os.Stat(filepath.Join(r.dir, "free")); !os.IsNotExist(err) {
		t.Fatal("untracked was carried")
	}
	oid := r.git("rev-parse", "stash@{0}")
	out := buf.String()
	if !strings.Contains(out, "same label") || !strings.Contains(out, oid) || !strings.Contains(out, "git stash apply --index "+oid) {
		t.Fatalf("out=%q", out)
	}
	if err := runSwitch(prompt.NewNonInteractive(), "main"); err != nil {
		t.Fatal(err)
	}
	if r.read("f") != "base\n" {
		t.Fatal("stash was applied on return")
	}
	r.write("f", "again\n")
	if err := recovery.Guard("work.list", false); err != nil {
		t.Fatal(err)
	}
	if refs := r.git("for-each-ref", "refs/elegant-git/recovery"); refs != "" {
		t.Fatal(refs)
	}
	if err := runStartOpt(prompt.NewNonInteractive(), "stash", "same label", "other"); err != nil {
		t.Fatal(err)
	}
	second := r.git("rev-parse", "stash@{0}")
	if second == oid || !strings.Contains(r.stashList(), "same label") {
		t.Fatalf("oid %s second %s list %q", oid, second, r.stashList())
	}
	if strings.Count(r.stashList(), "same label") < 2 {
		t.Fatal(r.stashList())
	}
}

func TestStashNamePrompt(t *testing.T) {
	t.Run("enter accepts", func(t *testing.T) {
		r := dirtyPair(t)
		r.enter()
		p := &swPrompter{picks: []string{"stash"}}
		buf := captureText(t)
		if err := runSwitch(p, "feat"); err != nil {
			t.Fatal(err)
		}
		if len(p.stringDefs) != 1 || !strings.HasPrefix(p.stringDefs[0], "WIP on main — ") {
			t.Fatalf("suggestion %v", p.stringDefs)
		}
		if !strings.Contains(buf.String(), p.stringDefs[0]) || !strings.Contains(r.stashList(), "WIP on main") {
			t.Fatalf("out %q stash %q", buf.String(), r.stashList())
		}
	})
	t.Run("edit", func(t *testing.T) {
		r := dirtyPair(t)
		r.enter()
		p := &swPrompter{picks: []string{"stash"}, lines: []string{"edited label"}}
		if err := runSwitch(p, "feat"); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(r.stashList(), "edited label") {
			t.Fatal(r.stashList())
		}
	})
	t.Run("cancel before stash", func(t *testing.T) {
		r := dirtyPair(t)
		r.enter()
		err := runSwitch(&swPrompter{picks: []string{"stash"}, stringErr: io.EOF}, "feat")
		if err == nil || !strings.Contains(err.Error(), "cancelled") {
			t.Fatalf("err=%v", err)
		}
		if r.headRef() != "refs/heads/main" || r.read("f") != "dirty\n" || r.stashList() != "" {
			t.Fatal("stashed after cancel")
		}
	})
	t.Run("picker cancel", func(t *testing.T) {
		r := dirtyPair(t)
		r.enter()
		err := runSwitch(&swPrompter{pickErr: prompt.ErrUserCancelled}, "feat")
		if err == nil || !strings.Contains(err.Error(), "cancelled") {
			t.Fatalf("err=%v", err)
		}
		if r.stashList() != "" || r.read("f") != "dirty\n" {
			t.Fatal("changed")
		}
	})
	t.Run("generated script name", func(t *testing.T) {
		r := dirtyPair(t)
		r.enter()
		if err := runSwitchOpt(prompt.NewNonInteractive(), "stash", "", "feat"); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(r.stashList(), "WIP on main") {
			t.Fatal(r.stashList())
		}
	})
	t.Run("rejected tokens", func(t *testing.T) {
		for _, token := range []string{"preserve", "fail", "add"} {
			r := dirtyPair(t)
			r.enter()
			err := runSwitchOpt(prompt.NewNonInteractive(), token, "", "feat")
			if err == nil || !strings.Contains(err.Error(), token) {
				t.Fatalf("%s err=%v", token, err)
			}
			if r.stashList() != "" || r.read("f") != "dirty\n" || r.headRef() != "refs/heads/main" {
				t.Fatal("mutated")
			}
		}
	})
}

func TestDirtyCurrentBranchIsNoop(t *testing.T) {
	r := dirtyPair(t)
	r.enter()
	if err := runSwitchOpt(prompt.NewNonInteractive(), "discard", "", "main"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/main" || r.read("f") != "dirty\n" || r.stashList() != "" {
		t.Fatal("current branch was prepared")
	}
}

func TestFailedHandoffKeepsRecovery(t *testing.T) {
	r := dirtyPair(t)
	r.enter()
	pipe.BeforeRetainHandoff = func() {
		r.git("stash", "clear")
	}
	t.Cleanup(func() { pipe.BeforeRetainHandoff = nil })
	err := runSwitchOpt(prompt.NewNonInteractive(), "stash", "kept", "feat")
	if err == nil || !strings.Contains(err.Error(), "eg repo doctor") {
		t.Fatalf("err=%v", err)
	}
	if err := recovery.Guard("work.list", false); err == nil {
		t.Fatal("recovery did not keep ownership")
	}
	if refs := r.git("for-each-ref", "refs/elegant-git/recovery"); refs == "" {
		t.Fatal("private ref was released")
	}
}

func TestStartRefusesMerge(t *testing.T) {
	r := newSwRepo(t)
	setupConflictMerge(r)
	r.enter()
	before := r.rev("HEAD")
	buf := captureText(t)
	err := runStartOpt(prompt.NewNonInteractive(), "carry", "", "other")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "merge") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.rev("HEAD") != before || r.gitAllow("rev-parse", "--verify", "--quiet", "refs/heads/other") == nil {
		t.Fatal("branch was created")
	}
}

func TestListDoesNotPrepareDirty(t *testing.T) {
	r := dirtyPair(t)
	r.enter()
	if err := listRun(io.Discard); err != nil {
		t.Fatal(err)
	}
	if r.read("f") != "dirty\n" || r.stashList() != "" || r.headRef() != "refs/heads/main" {
		t.Fatal("list prepared the tree")
	}
}

func TestCleanIgnoredObstruction(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "other")
	r.write("x.log", "tracked\n")
	r.git("add", "-f", "x.log")
	r.git("commit", "-q", "-m", "log")
	r.git("checkout", "-q", "main")
	r.write(".gitignore", "*.log\n")
	r.git("add", ".gitignore")
	r.git("commit", "-q", "-m", "ignore")
	r.write("x.log", "mine\n")
	r.enter()
	buf := captureText(t)
	err := runSwitch(prompt.NewNonInteractive(), "other")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "x.log") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.headRef() != "refs/heads/main" || r.read("x.log") != "mine\n" || r.stashList() != "" {
		t.Fatalf("head %s file %q stash %q", r.headRef(), r.read("x.log"), r.stashList())
	}
}

func TestBranchNamedLikeDirectory(t *testing.T) {
	r := newSwRepo(t)
	if err := os.MkdirAll(filepath.Join(r.dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.commit("docs", "docs/a", "a\n")
	r.git("branch", "docs")
	r.write("f", "dirty\n")
	r.enter()
	if err := runSwitchOpt(prompt.NewNonInteractive(), "carry", "", "docs"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/docs" || r.read("f") != "dirty\n" || r.read("docs/a") != "a\n" {
		t.Fatalf("head %s f %q", r.headRef(), r.read("f"))
	}
}

func TestSwitchSameNameTagUsesBranch(t *testing.T) {
	t.Run("branch tracks the ignored file", func(t *testing.T) {
		r := newSwRepo(t)
		r.git("checkout", "-q", "-b", "x")
		r.write("x.log", "t\n")
		r.git("add", "-f", "x.log")
		r.git("commit", "-q", "-m", "log")
		r.git("checkout", "-q", "main")
		r.git("tag", "x")
		r.write(".gitignore", "x.log\n")
		r.git("add", ".gitignore")
		r.git("commit", "-q", "-m", "ignore")
		r.write("x.log", "keep\n")
		r.enter()
		buf := captureText(t)
		err := runSwitch(prompt.NewNonInteractive(), "x")
		assertWorkflow(t, err)
		if !strings.Contains(buf.String(), "x.log") || r.headRef() != "refs/heads/main" || r.read("x.log") != "keep\n" {
			t.Fatalf("head %s file %q out %q", r.headRef(), r.read("x.log"), buf.String())
		}
	})
	t.Run("tag tracks the ignored file", func(t *testing.T) {
		r := newSwRepo(t)
		r.git("branch", "x")
		r.write("x.log", "tagged\n")
		r.git("add", "-f", "x.log")
		r.git("commit", "-q", "-m", "log")
		r.git("tag", "x")
		r.git("reset", "--hard", "-q", "HEAD~1")
		r.write(".gitignore", "x.log\n")
		r.git("add", ".gitignore")
		r.git("commit", "-q", "-m", "ignore")
		r.write("x.log", "keep\n")
		r.enter()
		if err := runSwitch(prompt.NewNonInteractive(), "x"); err != nil {
			t.Fatal(err)
		}
		if r.headRef() != "refs/heads/x" || r.read("x.log") != "keep\n" {
			t.Fatalf("head %s file %q", r.headRef(), r.read("x.log"))
		}
	})
}

func TestCarryRefusesUntrackedCollision(t *testing.T) {
	cases := []struct {
		name     string
		setup    func(*swRepo)
		occupant string
		read     string
	}{
		{
			name: "same file",
			setup: func(r *swRepo) {
				r.write("n.txt", "dest\n")
				r.git("add", "n.txt")
				r.git("commit", "-q", "-m", "n")
				r.git("checkout", "-q", "main")
				r.write("n.txt", "local\n")
			},
			occupant: "n.txt",
			read:     "n.txt",
		},
		{
			name: "file versus directory",
			setup: func(r *swRepo) {
				if err := os.MkdirAll(filepath.Join(r.dir, "nd"), 0o755); err != nil {
					t.Fatal(err)
				}
				r.write("nd/a.txt", "dest\n")
				r.git("add", "nd/a.txt")
				r.git("commit", "-q", "-m", "dir")
				r.git("checkout", "-q", "main")
				r.write("nd", "local\n")
			},
			occupant: "nd",
			read:     "nd",
		},
		{
			name: "directory versus file",
			setup: func(r *swRepo) {
				r.write("zz", "dest\n")
				r.git("add", "zz")
				r.git("commit", "-q", "-m", "file")
				r.git("checkout", "-q", "main")
				if err := os.MkdirAll(filepath.Join(r.dir, "zz"), 0o755); err != nil {
					t.Fatal(err)
				}
				r.write("zz/a.txt", "local\n")
			},
			occupant: "zz",
			read:     "zz/a.txt",
		},
		{
			name: "symlink versus directory",
			setup: func(r *swRepo) {
				if err := os.MkdirAll(filepath.Join(r.dir, "sl"), 0o755); err != nil {
					t.Fatal(err)
				}
				r.write("sl/a.txt", "dest\n")
				r.git("add", "sl/a.txt")
				r.git("commit", "-q", "-m", "dir")
				r.git("checkout", "-q", "main")
				if err := os.Symlink("somewhere", filepath.Join(r.dir, "sl")); err != nil {
					t.Fatal(err)
				}
			},
			occupant: "sl",
			read:     "sl",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newSwRepo(t)
			r.git("checkout", "-q", "-b", "other")
			tc.setup(r)
			r.write("f", "edit\n")
			r.enter()
			before := r.rev("HEAD")
			buf := captureText(t)
			err := runSwitchOpt(prompt.NewNonInteractive(), "carry", "", "other")
			assertWorkflow(t, err)
			if !strings.Contains(buf.String(), tc.occupant) || !strings.Contains(buf.String(), "left in place") {
				t.Fatalf("out=%q", buf.String())
			}
			if r.rev("HEAD") != before || r.read("f") != "edit\n" || r.stashList() != "" {
				t.Fatalf("head %s f %q stash %q", r.rev("HEAD"), r.read("f"), r.stashList())
			}
			if tc.name == "symlink versus directory" {
				info, statErr := os.Lstat(filepath.Join(r.dir, "sl"))
				if statErr != nil || info.Mode()&os.ModeSymlink == 0 {
					t.Fatalf("symlink %v", statErr)
				}
			} else if r.read(tc.read) == "" {
				t.Fatal("occupant lost")
			}
			if refs := r.git("for-each-ref", recovery.RefPrefix); refs != "" {
				t.Fatal(refs)
			}
		})
	}
}

func TestCarryRefusesCaseFoldCollision(t *testing.T) {
	r := newSwRepo(t)
	r.git("config", "core.ignorecase", "true")
	r.write("probe.txt", "x\n")
	if _, err := os.Lstat(filepath.Join(r.dir, "PROBE.txt")); err != nil {
		t.Skip("filesystem is case-sensitive")
	}
	if err := os.Remove(filepath.Join(r.dir, "probe.txt")); err != nil {
		t.Fatal(err)
	}
	r.git("checkout", "-q", "-b", "other")
	r.write("n.txt", "dest\n")
	r.git("add", "n.txt")
	r.git("commit", "-q", "-m", "n")
	r.git("checkout", "-q", "main")
	r.write("N.txt", "local\n")
	r.write("f", "edit\n")
	r.enter()
	before := r.rev("HEAD")
	buf := captureText(t)
	err := runSwitchOpt(prompt.NewNonInteractive(), "carry", "", "other")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "left in place") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.rev("HEAD") != before || r.read("N.txt") != "local\n" || r.read("f") != "edit\n" || r.stashList() != "" {
		t.Fatalf("head %s n %q f %q stash %q", r.rev("HEAD"), r.read("N.txt"), r.read("f"), r.stashList())
	}
}

func TestCarryRefusesUntrackedDespiteSameNameTag(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "other")
	r.write("n.txt", "dest\n")
	r.git("add", "n.txt")
	r.git("commit", "-q", "-m", "n")
	r.git("checkout", "-q", "main")
	r.git("tag", "other")
	r.write("n.txt", "local\n")
	r.write("f", "edit\n")
	r.enter()
	before := r.rev("HEAD")
	err := runSwitchOpt(prompt.NewNonInteractive(), "carry", "", "other")
	assertWorkflow(t, err)
	if r.rev("HEAD") != before || r.read("n.txt") != "local\n" || r.read("f") != "edit\n" || r.stashList() != "" {
		t.Fatalf("head %s n %q f %q stash %q", r.rev("HEAD"), r.read("n.txt"), r.read("f"), r.stashList())
	}
}

func TestStartDirtyFromTagRemoteAndSHA(t *testing.T) {
	for _, mode := range []string{"carry", "stash"} {
		t.Run("tag "+mode, func(t *testing.T) {
			r := newSwRepo(t)
			r.git("tag", "v1")
			r.write("f", "edit\n")
			r.enter()
			if err := runStartOpt(prompt.NewNonInteractive(), mode, stashLabel(mode), "feat", "v1"); err != nil {
				t.Fatal(err)
			}
			assertStarted(t, r, mode)
		})
		t.Run("upstream "+mode, func(t *testing.T) {
			r := newSwRepo(t)
			r.addRemote("origin", "main")
			r.write("f", "edit\n")
			r.enter()
			if err := runStartOpt(prompt.NewNonInteractive(), mode, stashLabel(mode), "feat"); err != nil {
				t.Fatal(err)
			}
			assertStarted(t, r, mode)
		})
		t.Run("sha "+mode, func(t *testing.T) {
			r := newSwRepo(t)
			sha := r.rev("HEAD")
			r.write("f", "edit\n")
			r.enter()
			cmd := &cobra.Command{}
			addDirtyFlags(cmd)
			cmd.SetContext(prompt.WithPrompter(context.Background(), prompt.NewNonInteractive()))
			if err := cmd.Flags().Set("dirty", mode); err != nil {
				t.Fatal(err)
			}
			if mode == "stash" {
				if err := cmd.Flags().Set("stash-name", "kept"); err != nil {
					t.Fatal(err)
				}
			}
			if err := startRunWithRefs(cmd, "feat", sha); err != nil {
				t.Fatal(err)
			}
			assertStarted(t, r, mode)
		})
	}
}

func stashLabel(mode string) string {
	if mode == "stash" {
		return "kept"
	}
	return ""
}

func assertStarted(t *testing.T, r *swRepo, mode string) {
	t.Helper()
	if r.headRef() != "refs/heads/feat" {
		t.Fatalf("head %s", r.headRef())
	}
	if mode == "carry" && r.read("f") != "edit\n" {
		t.Fatalf("file %q", r.read("f"))
	}
	if mode == "stash" && (r.read("f") != "base\n" || !strings.Contains(r.stashList(), "kept")) {
		t.Fatalf("file %q stash %q", r.read("f"), r.stashList())
	}
}

func TestCarryDetachedHookNamesOID(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "feat")
	r.git("checkout", "-q", "--detach")
	r.write("f", "edit\n")
	hook := filepath.Join(r.dir, ".git", "hooks", "post-checkout")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.enter()
	err := runSwitchOpt(prompt.NewNonInteractive(), "carry", "", "feat")
	if err == nil {
		t.Fatal("expected hook failure")
	}
	oid := strings.TrimSpace(r.git("rev-parse", "stash@{0}"))
	if oid == "" || !strings.Contains(err.Error(), oid) {
		t.Fatalf("err %v oid %s", err, oid)
	}
}

func TestCarryRefusesUntrackedOnStartFromRef(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "other")
	r.write("n.txt", "dest\n")
	r.git("add", "n.txt")
	r.git("commit", "-q", "-m", "n")
	r.git("checkout", "-q", "main")
	r.write("n.txt", "local\n")
	r.write("f", "edit\n")
	r.enter()
	before := r.rev("HEAD")
	buf := captureText(t)
	err := runStartOpt(prompt.NewNonInteractive(), "carry", "", "feat", "other")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "n.txt") {
		t.Fatalf("out=%q", buf.String())
	}
	if r.rev("HEAD") != before || r.read("n.txt") != "local\n" || r.read("f") != "edit\n" || r.stashList() != "" {
		t.Fatalf("head %s stash %q", r.rev("HEAD"), r.stashList())
	}
	if r.gitAllow("rev-parse", "--verify", "--quiet", "refs/heads/feat") == nil {
		t.Fatal("branch was created")
	}
}

func TestStashKeepsCollidingUntracked(t *testing.T) {
	r := newSwRepo(t)
	r.git("checkout", "-q", "-b", "other")
	r.write("n.txt", "dest\n")
	r.git("add", "n.txt")
	r.git("commit", "-q", "-m", "n")
	r.git("checkout", "-q", "main")
	r.write("n.txt", "local\n")
	r.write("f", "edit\n")
	r.enter()
	if err := runSwitchOpt(prompt.NewNonInteractive(), "stash", "kept", "other"); err != nil {
		t.Fatal(err)
	}
	if r.headRef() != "refs/heads/other" || r.read("n.txt") != "dest\n" || r.read("f") != "base\n" {
		t.Fatalf("head %s n %q f %q", r.headRef(), r.read("n.txt"), r.read("f"))
	}
	if got := r.git("show", "stash@{0}^3:n.txt"); got != "local" {
		t.Fatalf("stashed %q", got)
	}
	if refs := r.git("for-each-ref", recovery.RefPrefix); refs != "" {
		t.Fatal(refs)
	}
}

func TestCarryPostCheckoutHookNamesOID(t *testing.T) {
	r := dirtyPair(t)
	hook := filepath.Join(r.dir, ".git", "hooks", "post-checkout")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.enter()
	err := runSwitchOpt(prompt.NewNonInteractive(), "carry", "", "feat")
	if err == nil || !strings.Contains(err.Error(), "stay in the stash") {
		t.Fatal(err)
	}
	oid := strings.TrimSpace(r.git("rev-parse", "stash@{0}"))
	if oid == "" || !strings.Contains(err.Error(), oid) {
		t.Fatalf("err %v oid %s", err, oid)
	}
}

func TestDetachedCarryAndStash(t *testing.T) {
	t.Run("carry", func(t *testing.T) {
		r := newSwRepo(t)
		r.git("checkout", "-q", "--detach")
		r.write("f", "dirty\n")
		r.enter()
		if err := runStartOpt(prompt.NewNonInteractive(), "carry", "", "feat"); err != nil {
			t.Fatal(err)
		}
		if r.headRef() != "refs/heads/feat" || r.read("f") != "dirty\n" {
			t.Fatalf("head %s file %q", r.headRef(), r.read("f"))
		}
	})
	t.Run("stash", func(t *testing.T) {
		r := newSwRepo(t)
		r.git("checkout", "-q", "-b", "feat")
		r.git("checkout", "-q", "--detach")
		r.write("f", "dirty\n")
		r.enter()
		buf := captureText(t)
		if err := runSwitchOpt(prompt.NewNonInteractive(), "stash", "kept", "feat"); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		oid := strings.TrimSpace(r.git("rev-parse", "stash@{0}"))
		if r.headRef() != "refs/heads/feat" || r.read("f") != "base\n" || oid == "" || !strings.Contains(out, oid) {
			t.Fatalf("head %s file %q oid %s out %q", r.headRef(), r.read("f"), oid, out)
		}
		if !strings.Contains(out, "kept") || !strings.Contains(out, "stash@{") || !strings.Contains(out, "git stash apply --index "+oid) {
			t.Fatalf("out=%q", out)
		}
		if refs := r.git("for-each-ref", recovery.RefPrefix); refs != "" {
			t.Fatal(refs)
		}
	})
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
