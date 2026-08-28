package giturl

import "testing"

func TestNamespace(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"https://github.com/acme/app.git", "github.com/acme", true},
		{"https://user:token@github.com/acme/app.git", "github.com/acme", true},
		{"http://github.com/acme/app", "github.com/acme", true},
		{"git@github.com:acme/app.git", "github.com/acme", true},
		{"ssh://git@github.com:22/acme/app.git", "github.com/acme", true},
		{"ssh://git@github.com/acme/app.git", "github.com/acme", true},
		{"git://host/acme/app.git", "host/acme", true},
		{"https://gitlab.com/group/sub/app.git", "gitlab.com/group/sub", true},
		{"git@gitlab.com:group/sub/app.git", "gitlab.com/group/sub", true},
		{"HTTPS://GitHub.COM/Acme/App.GIT", "github.com/acme", true},
		{"https://github.com/acme/app.git/", "github.com/acme", true},
		{"/local/path", "", false},
		{"./relative", "", false},
		{"file:///tmp/repo.git", "", false},
		{"https://github.com/onlyrepo.git", "", false},
		{"https://github.com/", "", false},
		{"", "", false},
		{"   ", "", false},
	}
	for _, tt := range tests {
		got, ok := Namespace(tt.in)
		if ok != tt.ok || got != tt.want {
			t.Errorf("Namespace(%q) = (%q, %v); want (%q, %v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}
