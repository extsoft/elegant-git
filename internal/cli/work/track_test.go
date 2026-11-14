package work

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/prompt"
)

func TestRemoteBranchesMatching(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["for-each-ref --format=%(refname) refs/remotes"] = "refs/remotes/origin/rremote\nrefs/remotes/origin/master\nrefs/remotes/other-upstream/barr"
	git.Use(m)

	got := remoteBranchesMatching("rremote")
	if len(got) != 1 || got[0] != "origin/rremote" {
		t.Fatalf("got %+v", got)
	}
}

func TestTrackLogicPromptsAfterFetch(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/OPS-2114"
	m.Outputs["for-each-ref --format=%(refname) refs/remotes"] = "refs/remotes/origin/OPS-2114"
	git.Use(m)

	p := &trackRecordingPrompter{
		pickValues: []string{"origin/OPS-2114"},
		editValues: []string{""},
	}
	if err := trackLogic(context.Background(), p, "", ""); err != nil {
		t.Fatal(err)
	}
	if len(m.Calls) < 2 {
		t.Fatalf("calls=%v", m.Calls)
	}
	if m.Calls[0].Args[0] != "fetch" {
		t.Fatalf("first call=%v", m.Calls[0].Args)
	}
	fetchIdx := -1
	for i, c := range m.Calls {
		if len(c.Args) > 0 && c.Args[0] == "fetch" {
			fetchIdx = i
			break
		}
	}
	if fetchIdx < 0 {
		t.Fatal("missing fetch")
	}
	if len(p.stringDefaults) != 1 || p.stringDefaults[0] != "OPS-2114" || p.stringLabels[0] != "Local branch name" {
		t.Fatalf("strings=%v defaults=%v", p.stringLabels, p.stringDefaults)
	}
	checkoutIdx := -1
	for i, c := range m.Calls {
		if len(c.Args) > 0 && c.Args[0] == "checkout" {
			checkoutIdx = i
		}
	}
	if checkoutIdx <= fetchIdx {
		t.Fatalf("fetch=%d checkout=%d", fetchIdx, checkoutIdx)
	}
	if len(p.picks) != 1 {
		t.Fatalf("picks=%v", p.picks)
	}
	if !exactGitCall(m, "checkout", "-b", "OPS-2114", "--track", "refs/remotes/origin/OPS-2114") {
		t.Fatalf("calls=%v", m.Calls)
	}
}

func TestTrackLogicLocalBranchOverride(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["for-each-ref --format=%(refname) refs/remotes"] = "refs/remotes/origin/feature/133"
	git.Use(m)

	p := &trackRecordingPrompter{stringValues: []string{"task-133"}}
	if err := trackLogic(context.Background(), p, "origin/feature/133", ""); err != nil {
		t.Fatal(err)
	}
	if len(p.picks) != 0 {
		t.Fatalf("unexpected pick")
	}
	if len(p.stringDefaults) != 1 || p.stringDefaults[0] != "feature/133" || p.stringLabels[0] != "Local branch name" {
		t.Fatalf("strings=%v defaults=%v", p.stringLabels, p.stringDefaults)
	}
	if !exactGitCall(m, "checkout", "-b", "task-133", "--track", "refs/remotes/origin/feature/133") {
		t.Fatalf("calls=%v", m.Calls)
	}
	if hasGitCall(m, "checkout", "-B") {
		t.Fatalf("calls=%v", m.Calls)
	}
}

func TestTrackLogicNonInteractiveSkipsPrompts(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["for-each-ref --format=%(refname) refs/remotes"] = "refs/remotes/origin/feature/133"
	git.Use(m)

	p := prompt.NewNonInteractive()
	if err := trackLogic(context.Background(), p, "origin/feature/133", ""); err != nil {
		t.Fatal(err)
	}
	if !exactGitCall(m, "checkout", "-b", "feature/133", "--track", "refs/remotes/origin/feature/133") {
		t.Fatalf("calls=%v", m.Calls)
	}
	if hasGitCall(m, "checkout", "-B") {
		t.Fatalf("calls=%v", m.Calls)
	}
}

type trackRecordingPrompter struct {
	pickValues     []string
	editValues     []string
	stringValues   []string
	picks          []string
	pickChoices    [][]prompt.Choice
	pickDefaults   []string
	stringLabels   []string
	stringDefaults []string
	edits          []struct{ label, suggested string }
	pickIdx        int
	editIdx        int
	stringIdx      int
	onString       func()
}

func (p *trackRecordingPrompter) String(label, def string) (string, error) {
	if p.onString != nil {
		p.onString()
	}
	p.stringLabels = append(p.stringLabels, label)
	p.stringDefaults = append(p.stringDefaults, def)
	if p.stringIdx < len(p.stringValues) {
		v := p.stringValues[p.stringIdx]
		p.stringIdx++
		if v == "" {
			return def, nil
		}
		return v, nil
	}
	if def != "" {
		return def, nil
	}
	return "", fmt.Errorf("no more branch names")
}
func (p *trackRecordingPrompter) Confirm(string, bool) (bool, error) { return false, nil }
func (p *trackRecordingPrompter) Choose(string, []string) (int, error) {
	return -1, prompt.ErrNonInteractive
}
func (p *trackRecordingPrompter) Required(string, string) error { return nil }
func (p *trackRecordingPrompter) Optional(string, string) (string, error) {
	return "", nil
}
func (p *trackRecordingPrompter) Closed(string, []string, string, bool) (string, error) {
	return "", prompt.ErrNonInteractive
}
func (p *trackRecordingPrompter) BatchChoice(string, string) (prompt.BatchDecision, error) {
	return prompt.BatchSkip, nil
}

func (p *trackRecordingPrompter) Pick(label string, choices []prompt.Choice, def string) (string, error) {
	p.picks = append(p.picks, label)
	p.pickChoices = append(p.pickChoices, choices)
	p.pickDefaults = append(p.pickDefaults, def)
	if p.pickIdx < len(p.pickValues) {
		v := p.pickValues[p.pickIdx]
		p.pickIdx++
		return v, nil
	}
	return "", nil
}

func (p *trackRecordingPrompter) EditOrAccept(label, suggested string) (string, error) {
	p.edits = append(p.edits, struct{ label, suggested string }{label, suggested})
	if p.editIdx < len(p.editValues) {
		v := p.editValues[p.editIdx]
		p.editIdx++
		if v != "" {
			return v, nil
		}
	}
	return suggested, nil
}

func TestTrackForeignDuplicate(t *testing.T) {
	tests := []struct {
		name    string
		remote  string
		local   string
		heads   string
		upKey   string
		upValue string
		want    string
	}{
		{
			name:   "derived no upstream",
			remote: "origin/x",
			heads:  "x",
			want:   "branch 'x' already exists",
		},
		{
			name:    "cli other upstream",
			remote:  "origin/x",
			local:   "mine",
			heads:   "mine",
			upKey:   "rev-parse --symbolic-full-name mine@{upstream}",
			upValue: "refs/remotes/origin/elsewhere",
			want:    "branch 'mine' already exists",
		},
		{
			name:   "namespace",
			remote: "origin/feature/x",
			heads:  "feature",
			want:   "branch 'feature/x' conflicts with existing branch 'feature'",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := trackRunner(t, tc.remote, tc.heads)
			if tc.upKey != "" {
				m.Outputs[tc.upKey] = tc.upValue
			}
			buf := captureText(t)
			err := trackLogic(context.Background(), prompt.NewNonInteractive(), tc.remote, tc.local)
			assertWorkflow(t, err)
			if strings.Count(buf.String(), tc.want+"\n") != 1 {
				t.Fatalf("out=%q", buf.String())
			}
			assertNoTrackCheckout(t, m)
		})
	}
}

func TestTrackForeignDuplicateInteractive(t *testing.T) {
	t.Run("derived", func(t *testing.T) {
		m := trackRunner(t, "origin/x", "x")
		buf := captureText(t)
		p := &trackRecordingPrompter{stringValues: []string{"", "fresh"}}
		if err := trackLogic(context.Background(), p, "origin/x", ""); err != nil {
			t.Fatal(err)
		}
		if len(p.stringDefaults) < 1 || p.stringDefaults[0] != "x" || p.stringLabels[0] != "Local branch name" {
			t.Fatalf("strings=%v defaults=%v", p.stringLabels, p.stringDefaults)
		}
		if strings.Count(buf.String(), "branch 'x' already exists\n") != 1 {
			t.Fatalf("out=%q", buf.String())
		}
		if !exactGitCall(m, "checkout", "-b", "fresh", "--track", "refs/remotes/origin/x") {
			t.Fatalf("calls=%v", m.Calls)
		}
		assertNoForceCheckout(t, m)
	})
	t.Run("cli", func(t *testing.T) {
		m := trackRunner(t, "origin/x", "mine")
		m.Outputs["rev-parse --symbolic-full-name mine@{upstream}"] = "refs/remotes/origin/elsewhere"
		p := &trackRecordingPrompter{stringValues: []string{"", "fresh"}}
		if err := trackLogic(context.Background(), p, "origin/x", "mine"); err != nil {
			t.Fatal(err)
		}
		if len(p.stringDefaults) < 1 || p.stringDefaults[0] != "mine" || p.stringLabels[0] != "Local branch name" {
			t.Fatalf("strings=%v defaults=%v", p.stringLabels, p.stringDefaults)
		}
		if !exactGitCall(m, "checkout", "-b", "fresh", "--track", "refs/remotes/origin/x") {
			t.Fatalf("calls=%v", m.Calls)
		}
		assertNoForceCheckout(t, m)
	})
	t.Run("edit to a free name", func(t *testing.T) {
		m := trackRunner(t, "origin/x", "x")
		buf := captureText(t)
		p := &trackRecordingPrompter{stringValues: []string{"fresh"}}
		if err := trackLogic(context.Background(), p, "origin/x", ""); err != nil {
			t.Fatal(err)
		}
		if len(p.stringLabels) != 1 || p.stringLabels[0] != "Local branch name" || p.stringDefaults[0] != "x" {
			t.Fatalf("strings=%v defaults=%v", p.stringLabels, p.stringDefaults)
		}
		if strings.Contains(buf.String(), "already exists") {
			t.Fatalf("out=%q", buf.String())
		}
		if !exactGitCall(m, "checkout", "-b", "fresh", "--track", "refs/remotes/origin/x") {
			t.Fatalf("calls=%v", m.Calls)
		}
	})
}

func TestTrackSameUpstreamNonInteractive(t *testing.T) {
	m := trackRunner(t, "origin/x", "x")
	m.Outputs["rev-parse --symbolic-full-name x@{upstream}"] = "refs/remotes/origin/x"
	buf := captureText(t)
	err := trackLogic(context.Background(), prompt.NewNonInteractive(), "origin/x", "")
	assertWorkflow(t, err)
	want := "local branch 'x' already tracks 'origin/x'; run 'git switch x'\n"
	if strings.Count(buf.String(), want) != 1 {
		t.Fatalf("out=%q", buf.String())
	}
	assertNoTrackCheckout(t, m)
}

func TestTrackSameUpstreamChoices(t *testing.T) {
	t.Run("ancestor", func(t *testing.T) {
		m := trackSameUpstreamRepo(t)
		p := &trackRecordingPrompter{editValues: []string{""}, pickValues: []string{"switch"}}
		if err := trackLogic(context.Background(), p, "origin/x", ""); err != nil {
			t.Fatal(err)
		}
		assertUpstreamChoices(t, p, []string{"switch", "fast-forward", "reset", "new name"}, "fast-forward")
		reset, ok := choiceByValue(p.pickChoices[0], "reset")
		if !ok || reset.Description != "Point the local branch at the remote." {
			t.Fatalf("reset=%+v", reset)
		}
		if !exactGitCall(m, "checkout", "x", "--") {
			t.Fatalf("calls=%v", m.Calls)
		}
		assertNoForceCheckout(t, m)
	})
	t.Run("not ancestor", func(t *testing.T) {
		m := trackSameUpstreamRepo(t)
		m.FailOn["merge-base --is-ancestor refs/heads/x refs/remotes/origin/x"] = fmt.Errorf("exit status 1")
		m.Outputs["rev-list --count refs/remotes/origin/x..refs/heads/x"] = "2"
		p := &trackRecordingPrompter{editValues: []string{""}, pickValues: []string{"switch"}}
		if err := trackLogic(context.Background(), p, "origin/x", ""); err != nil {
			t.Fatal(err)
		}
		assertUpstreamChoices(t, p, []string{"switch", "reset", "new name"}, "switch")
		reset, ok := choiceByValue(p.pickChoices[0], "reset")
		if !ok || reset.Description != "Point the local branch at the remote and drop its other commits." {
			t.Fatalf("reset=%+v", reset)
		}
		if !exactGitCall(m, "checkout", "x", "--") {
			t.Fatalf("calls=%v", m.Calls)
		}
	})
	t.Run("fast-forward", func(t *testing.T) {
		m := trackSameUpstreamRepo(t)
		p := &trackRecordingPrompter{editValues: []string{""}, pickValues: []string{"fast-forward"}}
		if err := trackLogic(context.Background(), p, "origin/x", ""); err != nil {
			t.Fatal(err)
		}
		if !exactGitCall(m, "checkout", "x", "--") || !exactGitCall(m, "merge", "--ff-only", "refs/remotes/origin/x") {
			t.Fatalf("calls=%v", m.Calls)
		}
		assertNoForceCheckout(t, m)
	})
	t.Run("branch refs", func(t *testing.T) {
		m := trackSameUpstreamRepo(t)
		p := &trackRecordingPrompter{editValues: []string{""}, pickValues: []string{"switch"}}
		if err := trackLogic(context.Background(), p, "origin/x", ""); err != nil {
			t.Fatal(err)
		}
		if !exactGitCall(m, "merge-base", "--is-ancestor", "refs/heads/x", "refs/remotes/origin/x") {
			t.Fatalf("calls=%v", m.Calls)
		}
		if !exactGitCall(m, "rev-list", "--count", "refs/remotes/origin/x..refs/heads/x") {
			t.Fatalf("calls=%v", m.Calls)
		}
	})
	t.Run("rev-list error", func(t *testing.T) {
		m := trackSameUpstreamRepo(t)
		m.FailOn["rev-list --count refs/remotes/origin/x..refs/heads/x"] = fmt.Errorf("rev-list failed")
		p := &trackRecordingPrompter{editValues: []string{""}, pickValues: []string{"switch"}}
		if err := trackLogic(context.Background(), p, "origin/x", ""); err != nil {
			t.Fatal(err)
		}
		assertUpstreamChoices(t, p, []string{"switch", "fast-forward", "reset", "new name"}, "fast-forward")
		reset, ok := choiceByValue(p.pickChoices[0], "reset")
		if !ok || reset.Description != "Point the local branch at the remote and drop its other commits." {
			t.Fatalf("reset=%+v", reset)
		}
	})
	t.Run("rev-list unparsable", func(t *testing.T) {
		m := trackSameUpstreamRepo(t)
		m.Outputs["rev-list --count refs/remotes/origin/x..refs/heads/x"] = "nope"
		p := &trackRecordingPrompter{editValues: []string{""}, pickValues: []string{"switch"}}
		if err := trackLogic(context.Background(), p, "origin/x", ""); err != nil {
			t.Fatal(err)
		}
		reset, ok := choiceByValue(p.pickChoices[0], "reset")
		if !ok || reset.Description != "Point the local branch at the remote and drop its other commits." {
			t.Fatalf("reset=%+v", reset)
		}
	})
	t.Run("ancestor probe error", func(t *testing.T) {
		m := trackSameUpstreamRepo(t)
		m.FailOn["merge-base --is-ancestor refs/heads/x refs/remotes/origin/x"] = fmt.Errorf("not a git repository")
		p := &trackRecordingPrompter{editValues: []string{""}, pickValues: []string{"switch"}}
		if err := trackLogic(context.Background(), p, "origin/x", ""); err != nil {
			t.Fatal(err)
		}
		assertUpstreamChoices(t, p, []string{"switch", "reset", "new name"}, "switch")
	})
	t.Run("reset", func(t *testing.T) {
		m := trackSameUpstreamRepo(t)
		p := &trackRecordingPrompter{editValues: []string{""}, pickValues: []string{"reset"}}
		if err := trackLogic(context.Background(), p, "origin/x", ""); err != nil {
			t.Fatal(err)
		}
		if !exactGitCall(m, "checkout", "-B", "x", "refs/remotes/origin/x") {
			t.Fatalf("calls=%v", m.Calls)
		}
		if hasGitCall(m, "merge") {
			t.Fatalf("calls=%v", m.Calls)
		}
	})
	t.Run("new name", func(t *testing.T) {
		m := trackSameUpstreamRepo(t)
		var configBeforeNewName int
		stringsSeen := 0
		p := &trackRecordingPrompter{
			pickValues:   []string{"new name"},
			stringValues: []string{"", "fresh"},
			onString: func() {
				stringsSeen++
				if stringsSeen == 2 {
					configBeforeNewName = 0
					for _, c := range m.Calls {
						if strings.Join(c.Args, " ") == "config --get core.ignorecase" {
							configBeforeNewName++
						}
					}
				}
			},
		}
		if err := trackLogic(context.Background(), p, "origin/x", ""); err != nil {
			t.Fatal(err)
		}
		if configBeforeNewName != 1 {
			t.Fatalf("core.ignorecase reads before new-name prompt = %d", configBeforeNewName)
		}
		if len(p.stringLabels) != 2 || p.stringLabels[0] != "Local branch name" || p.stringLabels[1] != "Branch name" {
			t.Fatalf("strings=%v", p.stringLabels)
		}
		if !exactGitCall(m, "checkout", "-b", "fresh", "--track", "refs/remotes/origin/x") {
			t.Fatalf("calls=%v", m.Calls)
		}
		assertNoForceCheckout(t, m)
	})
}

func TestTrackBranchListError(t *testing.T) {
	m := trackRunner(t, "origin/x", "")
	m.FailOn["for-each-ref --format=%(refname:lstrip=2) refs/heads"] = fmt.Errorf("not a git repository")
	err := trackLogic(context.Background(), prompt.NewNonInteractive(), "origin/x", "")
	if err == nil || err.Error() != "not a git repository" {
		t.Fatalf("err=%v", err)
	}
	assertNoTrackCheckout(t, m)
}

func TestTrackExactRemoteStillResets(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["for-each-ref --format=%(refname:short) refs/remotes"] = "origin/feat"
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	if err := trackExactRemote("origin/feat", acceptWorkBranch); err != nil {
		t.Fatal(err)
	}
	if !exactGitCall(m, "checkout", "-B", acceptWorkBranch, "origin/feat") {
		t.Fatalf("calls=%v", m.Calls)
	}
}

func TestTrackRealRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	root := t.TempDir()
	bare := filepath.Join(root, "bare.git")
	clone := filepath.Join(root, "clone")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(root, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_EDITOR", "true")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
	if err := os.WriteFile(filepath.Join(root, "gitconfig"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command(git.Executable(), args...)
		cmd.Dir = dir
		env := make([]string, 0, len(os.Environ()))
		for _, e := range os.Environ() {
			if strings.HasPrefix(e, "GIT_DIR=") || strings.HasPrefix(e, "GIT_WORK_TREE=") {
				continue
			}
			env = append(env, e)
		}
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	gitAt(root, "init", "--bare", "-q", "-b", "main", bare)
	gitAt(root, "init", "-q", "-b", "main", clone)
	if err := os.WriteFile(filepath.Join(clone, "f"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(clone, "config", "user.email", "test@example.com")
	gitAt(clone, "config", "user.name", "Test")
	gitAt(clone, "config", "commit.gpgsign", "false")
	gitAt(clone, "add", "f")
	gitAt(clone, "commit", "-q", "-m", "base")
	gitAt(clone, "remote", "add", "origin", bare)
	gitAt(clone, "push", "-q", "-u", "origin", "main")
	gitAt(clone, "checkout", "-q", "-b", "topic")
	if err := os.WriteFile(filepath.Join(clone, "f"), []byte("topic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(clone, "commit", "-q", "-am", "topic")
	gitAt(clone, "push", "-q", "-u", "origin", "topic")
	gitAt(clone, "checkout", "-q", "main")
	gitAt(clone, "branch", "-D", "topic")
	gitAt(clone, "tag", "origin/topic", "main")
	gitAt(clone, "tag", "topic", "main")
	tagTopic := gitAt(clone, "rev-parse", "refs/tags/topic")
	tagRemote := gitAt(clone, "rev-parse", "refs/tags/origin/topic")

	t.Setenv("GIT_DIR", filepath.Join(clone, ".git"))
	t.Setenv("GIT_WORK_TREE", clone)
	git.Use(git.RealRunner{})
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	buf := captureText(t)
	head := func() string { return gitAt(clone, "rev-parse", "refs/heads/topic") }
	remoteTip := func() string { return gitAt(clone, "rev-parse", "refs/remotes/origin/topic") }

	buf.Reset()
	if err := trackLogic(context.Background(), prompt.NewNonInteractive(), "origin/topic", ""); err != nil {
		t.Fatal(err)
	}
	if head() != remoteTip() || head() == tagRemote {
		t.Fatalf("tip=%s remote=%s tag=%s", head(), remoteTip(), tagRemote)
	}
	if gitAt(clone, "config", "--get", "branch.topic.remote") != "origin" {
		t.Fatal(gitAt(clone, "config", "--get", "branch.topic.remote"))
	}
	if gitAt(clone, "config", "--get", "branch.topic.merge") != "refs/heads/topic" {
		t.Fatal(gitAt(clone, "config", "--get", "branch.topic.merge"))
	}
	if gitAt(clone, "rev-parse", "refs/tags/topic") != tagTopic {
		t.Fatal("tag topic moved")
	}

	tip := head()
	sym := gitAt(clone, "symbolic-ref", "HEAD")
	buf.Reset()
	err := trackLogic(context.Background(), prompt.NewNonInteractive(), "origin/topic", "")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "local branch 'topic' already tracks 'origin/topic'; run 'git switch topic'\n") {
		t.Fatalf("out=%q", buf.String())
	}
	if head() != tip || gitAt(clone, "symbolic-ref", "HEAD") != sym {
		t.Fatalf("moved tip=%s head=%s", head(), gitAt(clone, "symbolic-ref", "HEAD"))
	}

	gitAt(clone, "branch", "other", "main")
	other := gitAt(clone, "rev-parse", "refs/heads/other")
	tip = head()
	buf.Reset()
	err = trackLogic(context.Background(), prompt.NewNonInteractive(), "origin/topic", "other")
	assertWorkflow(t, err)
	if !strings.Contains(buf.String(), "branch 'other' already exists\n") {
		t.Fatalf("out=%q", buf.String())
	}
	if gitAt(clone, "rev-parse", "refs/heads/other") != other || head() != tip {
		t.Fatal("refs moved")
	}

	gitAt(clone, "checkout", "-q", "topic", "--")
	if err := os.WriteFile(filepath.Join(clone, "f"), []byte("ahead\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(clone, "commit", "-q", "-am", "ahead")
	gitAt(clone, "push", "-q", "origin", "refs/heads/topic:refs/heads/topic")
	gitAt(clone, "reset", "-q", "--hard", "HEAD~1")
	behind := head()
	p := &trackRecordingPrompter{editValues: []string{""}, pickValues: []string{"fast-forward"}}
	buf.Reset()
	if err := trackLogic(context.Background(), p, "origin/topic", ""); err != nil {
		t.Fatal(err)
	}
	if head() == behind || head() != remoteTip() || head() == tagRemote {
		t.Fatalf("tip=%s behind=%s remote=%s tag=%s", head(), behind, remoteTip(), tagRemote)
	}

	gitAt(clone, "checkout", "-q", "topic", "--")
	if err := os.WriteFile(filepath.Join(clone, "f"), []byte("local-only\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAt(clone, "commit", "-q", "-am", "local-only")
	tip = head()
	p = &trackRecordingPrompter{editValues: []string{""}, pickValues: []string{"switch"}}
	buf.Reset()
	if err := trackLogic(context.Background(), p, "origin/topic", ""); err != nil {
		t.Fatal(err)
	}
	reset, ok := choiceByValue(p.pickChoices[0], "reset")
	if !ok || reset.Description != "Point the local branch at the remote and drop its other commits." {
		t.Fatalf("reset=%+v", reset)
	}
	if head() != tip || gitAt(clone, "symbolic-ref", "HEAD") != "refs/heads/topic" {
		t.Fatalf("tip=%s head=%s", head(), gitAt(clone, "symbolic-ref", "HEAD"))
	}
}

func trackRunner(t *testing.T, remote, heads string) *git.MemoryRunner {
	t.Helper()
	m := git.NewMemoryRunner()
	m.Outputs["for-each-ref --format=%(refname) refs/remotes"] = "refs/remotes/" + remote
	if heads != "" {
		m.Outputs["for-each-ref --format=%(refname:lstrip=2) refs/heads"] = heads
	}
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })
	return m
}

func trackSameUpstreamRepo(t *testing.T) *git.MemoryRunner {
	t.Helper()
	m := trackRunner(t, "origin/x", "x")
	m.Outputs["rev-parse --symbolic-full-name x@{upstream}"] = "refs/remotes/origin/x"
	m.Outputs["rev-list --count refs/remotes/origin/x..refs/heads/x"] = "0"
	m.Outputs["for-each-ref --format="+branchUpstreamFormat+" refs/heads"] = "x\trefs/remotes/origin/x"
	return m
}

func TestTrackingSetMatchesFullUpstream(t *testing.T) {
	m := git.NewMemoryRunner()
	m.Outputs["for-each-ref --format="+branchUpstreamFormat+" refs/heads"] = strings.Join([]string{
		"x\trefs/remotes/origin/x",
		"mine\trefs/remotes/origin/elsewhere",
		"local\trefs/heads/main",
		"none\t",
		"plain",
	}, "\n")
	git.Use(m)
	t.Cleanup(func() { git.Use(git.RealRunner{}) })

	set := trackingSet("origin/x")
	if len(m.Calls) != 1 || strings.Join(m.Calls[0].Args, " ") != "for-each-ref --format="+branchUpstreamFormat+" refs/heads" {
		t.Fatalf("calls=%v", m.Calls)
	}
	if !set["x"] || set["mine"] || set["local"] || set["none"] || set["plain"] {
		t.Fatalf("set=%v", set)
	}
}

func assertUpstreamChoices(t *testing.T, p *trackRecordingPrompter, want []string, def string) {
	t.Helper()
	if len(p.picks) != 1 || p.picks[0] != "Local branch 'x' already tracks 'origin/x'" {
		t.Fatalf("picks=%v", p.picks)
	}
	if len(p.pickDefaults) != 1 || p.pickDefaults[0] != def {
		t.Fatalf("default=%v", p.pickDefaults)
	}
	got := make([]string, len(p.pickChoices[0]))
	for i, c := range p.pickChoices[0] {
		got[i] = c.Value
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("choices=%v", got)
	}
}

func choiceByValue(choices []prompt.Choice, value string) (prompt.Choice, bool) {
	for _, c := range choices {
		if c.Value == value {
			return c, true
		}
	}
	return prompt.Choice{}, false
}

func exactGitCall(m *git.MemoryRunner, args ...string) bool {
	for _, c := range m.Calls {
		if len(c.Args) != len(args) {
			continue
		}
		match := true
		for i := range args {
			if c.Args[i] != args[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func assertNoTrackCheckout(t *testing.T, m *git.MemoryRunner) {
	t.Helper()
	for _, c := range m.Calls {
		if len(c.Args) == 0 {
			continue
		}
		switch c.Args[0] {
		case "checkout", "merge", "reset":
			t.Fatalf("unexpected git %v", c.Args)
		}
	}
}

func assertNoForceCheckout(t *testing.T, m *git.MemoryRunner) {
	t.Helper()
	if hasGitCall(m, "checkout", "-B") {
		t.Fatalf("calls=%v", m.Calls)
	}
}
