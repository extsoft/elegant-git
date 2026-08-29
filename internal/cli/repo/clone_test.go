package repo

import (
	"path/filepath"
	"testing"
)

func TestDefaultCloneDir(t *testing.T) {
	cases := []struct {
		repo string
		want string
	}{
		{"git@github.com:Zwirner/active-data-pipelines", "active-data-pipelines"},
		{"git@github.com:extsoft/elegant-git.git", "elegant-git"},
		{"https://github.com/extsoft/elegant-git.git", "elegant-git"},
		{"https://github.com/extsoft/elegant-git.git/", "elegant-git"},
		{"ssh://git@github.com/user/repo.git", "repo"},
		{"/tmp/local/my-repo.git", "my-repo"},
		{"my-repo", "my-repo"},
	}
	for _, tc := range cases {
		if got := defaultCloneDir(tc.repo); got != tc.want {
			t.Errorf("defaultCloneDir(%q) = %q, want %q", tc.repo, got, tc.want)
		}
	}
}

func TestResolveUnderGitPrefix(t *testing.T) {
	t.Setenv("GIT_PREFIX", "")
	if got := resolveUnderGitPrefix("mydir"); got != "mydir" {
		t.Fatalf("empty prefix: %q", got)
	}

	t.Setenv("GIT_PREFIX", "docs/")
	if got := resolveUnderGitPrefix("mydir"); got != filepath.Join("docs/", "mydir") {
		t.Fatalf("relative: %q", got)
	}

	abs := filepath.Join(t.TempDir(), "out")
	if got := resolveUnderGitPrefix(abs); got != abs {
		t.Fatalf("abs: %q", got)
	}
}
