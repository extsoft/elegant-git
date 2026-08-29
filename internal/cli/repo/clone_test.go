package repo

import "testing"

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
