package work

import "testing"

func TestPushRemoteBranch(t *testing.T) {
	tests := []struct {
		local, arg, upstream, want string
	}{
		{"feature1", "", "", "feature1"},
		{"feature1", "", "origin/feature1", "feature1"},
		{"feature1", "other", "", "other"},
		{"OPS-2162", "", "origin/main", "OPS-2162"},
		{"123", "", "some-remote/feature/123", "feature/123"},
	}
	for _, tc := range tests {
		got := pushRemoteBranch(tc.local, tc.arg, tc.upstream)
		if got != tc.want {
			t.Errorf("pushRemoteBranch(%q, %q, %q) = %q, want %q", tc.local, tc.arg, tc.upstream, got, tc.want)
		}
	}
}
