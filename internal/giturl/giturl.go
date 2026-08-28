// Package giturl normalizes git remote URLs into workspace namespaces.
package giturl

import (
	"net/url"
	"strings"
)

// Namespace returns a lowercase <host>/<owner...> value derived from a git
// remote URL. Owner is every path segment except the last (the repository
// name). Local paths and file:// URLs return ("", false).
func Namespace(rawURL string) (string, bool) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", false
	}
	host, path, ok := parseRemote(rawURL)
	if !ok {
		return "", false
	}
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return "", false
	}
	path = strings.Trim(path, "/")
	path = strings.TrimSuffix(path, ".git")
	path = strings.Trim(path, "/")
	if path == "" {
		return "", false
	}
	segs := strings.Split(path, "/")
	if len(segs) < 2 {
		return "", false
	}
	owner := strings.Join(segs[:len(segs)-1], "/")
	owner = strings.ToLower(owner)
	if owner == "" {
		return "", false
	}
	return host + "/" + owner, true
}

func parseRemote(raw string) (host, path string, ok bool) {
	if strings.HasPrefix(raw, "file://") || strings.HasPrefix(raw, "/") ||
		strings.HasPrefix(raw, "./") || strings.HasPrefix(raw, "../") ||
		(len(raw) > 1 && raw[1] == ':') { // Windows drive path
		return "", "", false
	}
	// SCP-like: git@github.com:acme/app.git
	if !strings.Contains(raw, "://") {
		if at := strings.Index(raw, "@"); at >= 0 {
			rest := raw[at+1:]
			if colon := strings.Index(rest, ":"); colon >= 0 {
				host = rest[:colon]
				path = rest[colon+1:]
				return host, path, true
			}
		}
		// Bare path without scheme — treat as local.
		return "", "", false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", "", false
	}
	if u.Scheme == "file" {
		return "", "", false
	}
	host = u.Hostname() // strips port
	path = u.Path
	return host, path, true
}
