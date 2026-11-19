package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// MinVersion is the oldest Git release that can run Git-backed Elegant Git commands.
// It is a functional minimum, not a security recommendation.
const MinVersion = "2.40.0"

type parsedVersion struct {
	major, minor, patch int
	stable              bool
}

var minimumVersion = mustParseToken(MinVersion)

func mustParseToken(token string) parsedVersion {
	v, err := parseToken(token)
	if err != nil || !v.stable {
		panic(err)
	}
	return v
}

var (
	compatMu     sync.Mutex
	versionProbe = probeExecutable
	compatCache  = map[string]error{}
	compatSeen   = map[string]bool{}
)

// SetVersionProbeForTest replaces real `git version` discovery and clears the process cache.
// A nil probe restores the executable probe. Injected runners do not use this cache.
func SetVersionProbeForTest(probe func(exe string) (string, error)) {
	compatMu.Lock()
	defer compatMu.Unlock()
	if probe == nil {
		versionProbe = probeExecutable
	} else {
		versionProbe = probe
	}
	compatCache = map[string]error{}
	compatSeen = map[string]bool{}
}

// Check reports whether the selected Git executable meets MinVersion.
// Real discovery is cached for the executable for this process. An injected runner
// is read from its own `version` output and is not cached.
func Check() error {
	if _, ok := current.(RealRunner); !ok && current != nil {
		out, err := current.Output("version")
		return decide(Executable(), out, err)
	}
	return checkReal()
}

func checkReal() error {
	return checkExe(Executable())
}

func checkExe(exe string) error {
	compatMu.Lock()
	if compatSeen[exe] {
		err := compatCache[exe]
		compatMu.Unlock()
		return err
	}
	probe := versionProbe
	compatMu.Unlock()

	out, err := probe(exe)
	decided := decide(exe, out, err)

	compatMu.Lock()
	defer compatMu.Unlock()
	if compatSeen[exe] {
		return compatCache[exe]
	}
	compatSeen[exe] = true
	compatCache[exe] = decided
	return decided
}

func probeExecutable(exe string) (string, error) {
	cmd := exec.Command(exe, "version")
	out, err := cmd.Output()
	return string(out), err
}

func selectedExecutable(exe string) string {
	if exe == "" || strings.ContainsRune(exe, os.PathSeparator) || strings.Contains(exe, "/") {
		return exe
	}
	path, err := exec.LookPath(exe)
	if err != nil || path == "" {
		return exe
	}
	return path
}

func decide(exe, out string, runErr error) error {
	exe = selectedExecutable(exe)
	line := firstVersionLine(out)
	if runErr != nil {
		return incompatible(exe, line, runErr, false)
	}
	token, _, err := versionToken(line)
	if err != nil {
		return incompatible(exe, line, err, false)
	}
	parsed, err := parseToken(token)
	if err != nil {
		return incompatible(exe, line, err, false)
	}
	if !parsed.stable {
		return incompatible(exe, line, nil, true)
	}
	if compareVersion(parsed, minimumVersion) < 0 {
		return incompatible(exe, line, nil, false)
	}
	return nil
}

func incompatible(exe, observed string, cause error, prerelease bool) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Elegant Git requires Git %s or newer", MinVersion)
	switch {
	case prerelease:
		fmt.Fprintf(&b, "; this Git is %s, which is not a stable release", oneLine(observed))
	case observed != "":
		fmt.Fprintf(&b, "; this Git is %s", oneLine(observed))
	case cause != nil:
		fmt.Fprintf(&b, "; git version failed: %s", cause)
	default:
		b.WriteString("; git version failed")
	}
	fmt.Fprintf(&b, " (%s). ", exe)
	b.WriteString("Install a maintained Git release and put it first on PATH. ")
	fmt.Fprintf(&b, "Git %s is the oldest version that can run these commands. It is not a security recommendation.", MinVersion)
	return errors.New(b.String())
}

func oneLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

func firstVersionLine(out string) string {
	var first string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" {
			continue
		}
		if first == "" {
			first = line
		}
		if strings.HasPrefix(line, "git version ") {
			return line
		}
	}
	return first
}

func versionToken(line string) (token, label string, err error) {
	const prefix = "git version "
	if !strings.HasPrefix(line, prefix) {
		return "", "", fmt.Errorf("unrecognized git version %q", line)
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	if rest == "" {
		return "", "", fmt.Errorf("unrecognized git version %q", line)
	}
	token, label, ok := strings.Cut(rest, " ")
	if !ok {
		return token, "", nil
	}
	label = strings.TrimSpace(label)
	if !strings.HasPrefix(label, "(") || !strings.HasSuffix(label, ")") {
		return "", "", fmt.Errorf("unrecognized git version %q", line)
	}
	return token, label, nil
}

func parseToken(token string) (parsedVersion, error) {
	parts := strings.Split(token, ".")
	if len(parts) == 3 && allDigits(parts[0]) && allDigits(parts[1]) && allDigits(parts[2]) {
		return parseTriple(parts[0], parts[1], parts[2], true)
	}
	if len(parts) == 5 && allDigits(parts[0]) && allDigits(parts[1]) && allDigits(parts[2]) && vendorName(parts[3]) && allDigits(parts[4]) {
		return parseTriple(parts[0], parts[1], parts[2], true)
	}
	base, rest, ok := strings.Cut(token, "-")
	if ok && preID(rest) {
		baseParts := strings.Split(base, ".")
		if len(baseParts) == 3 && allDigits(baseParts[0]) && allDigits(baseParts[1]) && allDigits(baseParts[2]) {
			return parseTriple(baseParts[0], baseParts[1], baseParts[2], false)
		}
	}
	return parsedVersion{}, fmt.Errorf("unrecognized git version %q", token)
}

func parseTriple(major, minor, patch string, stable bool) (parsedVersion, error) {
	maj, err := strconv.Atoi(major)
	if err != nil {
		return parsedVersion{}, err
	}
	min, err := strconv.Atoi(minor)
	if err != nil {
		return parsedVersion{}, err
	}
	pat, err := strconv.Atoi(patch)
	if err != nil {
		return parsedVersion{}, err
	}
	return parsedVersion{major: maj, minor: min, patch: pat, stable: stable}, nil
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func vendorName(s string) bool {
	if s == "" || !isLetter(rune(s[0])) {
		return false
	}
	for _, c := range s {
		if !isLetter(c) && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func isLetter(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func preID(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if isLetter(c) || (c >= '0' && c <= '9') || c == '.' || c == '-' {
			continue
		}
		return false
	}
	return true
}

func compareVersion(a, b parsedVersion) int {
	if a.major != b.major {
		return cmpInt(a.major, b.major)
	}
	if a.minor != b.minor {
		return cmpInt(a.minor, b.minor)
	}
	return cmpInt(a.patch, b.patch)
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
