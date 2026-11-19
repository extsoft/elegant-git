package work

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

const changedBeforePrep = "The working tree changed before preparation. Nothing was stashed, reset, or deleted. Run the command again to preview it."

const noDiscardBackup = "No discard backup exists."

type dirtyFlags struct {
	choice       string
	name         string
	nameSet      bool
	untracked    string
	untrackedSet bool
}

type filePrint struct {
	mode    os.FileMode
	size    int64
	mod     int64
	sum     string
	missing bool
}

type newPath struct {
	path  string
	kind  string
	print filePrint
	block string
}

type workIdentity struct {
	head   string
	index  string
	status string
}

type discardSet struct {
	id       workIdentity
	cands    []newPath
	unstaged []string
	restore  []string
}

type discardAuth struct {
	plan discardSet
	disp string
}

type discardAuthKey struct{}

func showDirtyPreview(dest string) error {
	snap, err := state.ReadSnapshot()
	if err != nil {
		return err
	}
	if !snap.Relevant() {
		return nil
	}
	if dest == "" {
		dest = "the destination"
	}
	text.PlainText("Before " + dest + ":")
	writeStatusGroup("Staged", stagedStatus(snap.Lines))
	writeStatusGroup("Unstaged", unstagedStatus(snap.Lines))
	writeStatusGroup("Untracked", untrackedStatus(snap.Lines))
	writeStatusGroup("Unmerged", unmergedStatus(snap.Lines))
	return nil
}

func writeStatusGroup(title string, lines []string) {
	if len(lines) == 0 {
		return
	}
	text.PlainText(title + ":")
	for _, line := range lines {
		text.PlainText(line)
	}
}

func stagedStatus(lines []state.StatusLine) []string {
	var out []string
	for _, ln := range lines {
		if isUnmergedXY(ln.X, ln.Y) || ln.X == ' ' || ln.X == '?' || ln.X == '!' {
			continue
		}
		out = append(out, fmt.Sprintf("  %-10s %s", statusLabel(ln.X), strconv.Quote(ln.Path)))
	}
	return out
}

func unstagedStatus(lines []state.StatusLine) []string {
	var out []string
	for _, ln := range lines {
		if isUnmergedXY(ln.X, ln.Y) || ln.Y == ' ' || ln.Y == '?' || ln.Y == '!' {
			continue
		}
		out = append(out, fmt.Sprintf("  %-10s %s", statusLabel(ln.Y), strconv.Quote(ln.Path)))
	}
	return out
}

func untrackedStatus(lines []state.StatusLine) []string {
	var out []string
	for _, ln := range lines {
		if ln.X == '?' && ln.Y == '?' {
			out = append(out, "  "+strconv.Quote(ln.Path))
		}
	}
	return out
}

func unmergedStatus(lines []state.StatusLine) []string {
	var out []string
	for _, ln := range lines {
		if isUnmergedXY(ln.X, ln.Y) {
			out = append(out, "  "+strconv.Quote(ln.Path))
		}
	}
	return out
}

func statusLabel(c byte) string {
	switch c {
	case 'M':
		return "modified"
	case 'A':
		return "added"
	case 'D':
		return "deleted"
	case 'R':
		return "renamed"
	case 'C':
		return "copied"
	case 'T':
		return "typechange"
	case 'U':
		return "unmerged"
	default:
		return "changed"
	}
}

func isUnmergedXY(x, y byte) bool {
	switch string([]byte{x, y}) {
	case "DD", "AU", "UD", "UA", "DU", "AA", "UU":
		return true
	default:
		return false
	}
}

func readIdentity() (workIdentity, error) {
	headOut, headErr := git.Output("rev-parse", "--verify", "--quiet", "HEAD")
	head := strings.TrimSpace(headOut)
	if headErr != nil {
		if !state.HeadUnborn() {
			return workIdentity{}, headErr
		}
		head = ""
	}
	index, err := indexSum()
	if err != nil {
		return workIdentity{}, err
	}
	status, err := git.Output(state.StatusArgs()...)
	if err != nil {
		msg := strings.TrimSpace(status)
		if msg == "" {
			msg = err.Error()
		}
		return workIdentity{}, fmt.Errorf("%s", msg)
	}
	return workIdentity{head: head, index: index, status: status}, nil
}

func indexSum() (string, error) {
	p, err := git.Output("rev-parse", "--git-path", "index")
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(strings.TrimSpace(p))
	if err != nil {
		if os.IsNotExist(err) && state.HeadUnborn() {
			return "", nil
		}
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func (a workIdentity) same(b workIdentity) bool {
	return a.head == b.head && a.index == b.index && a.status == b.status
}

func (p discardSet) same(o discardSet) bool {
	if !p.id.same(o.id) || len(p.cands) != len(o.cands) || len(p.unstaged) != len(o.unstaged) || len(p.restore) != len(o.restore) {
		return false
	}
	for i := range p.cands {
		if p.cands[i] != o.cands[i] {
			return false
		}
	}
	for i := range p.unstaged {
		if p.unstaged[i] != o.unstaged[i] {
			return false
		}
	}
	for i := range p.restore {
		if p.restore[i] != o.restore[i] {
			return false
		}
	}
	return true
}

func planTracked() (keep, restore []string, kinds map[string]string, err error) {
	kinds = map[string]string{}
	for _, cached := range []bool{false, true} {
		args := []string{"diff", "-z", "--name-status", "--find-renames", "HEAD"}
		if cached {
			args = []string{"diff", "-z", "--cached", "--name-status", "--find-renames", "HEAD"}
		}
		out, err := git.Output(args...)
		if err != nil {
			msg := strings.TrimSpace(out)
			if msg == "" {
				msg = err.Error()
			}
			return nil, nil, nil, fmt.Errorf("%s", msg)
		}
		k, r, kind, err := classifyNameStatus(out)
		if err != nil {
			return nil, nil, nil, err
		}
		keep = append(keep, k...)
		restore = append(restore, r...)
		for path, label := range kind {
			kinds[path] = label
		}
	}
	return keep, restore, kinds, nil
}

func buildDiscardPlan() (discardSet, error) {
	keep, restore, kinds, err := planTracked()
	if err != nil {
		return discardSet{}, err
	}
	snap, err := state.ReadSnapshot()
	if err != nil {
		return discardSet{}, err
	}
	inHead, err := pathsInHEAD(keep)
	if err != nil {
		return discardSet{}, err
	}
	var unstaged []string
	for _, path := range keep {
		if inHead[path] {
			restore = append(restore, path)
			continue
		}
		unstaged = append(unstaged, path)
	}
	unstaged = uniquePaths(unstaged)
	restore = uniquePaths(restore)
	root, err := repoRoot()
	if err != nil {
		return discardSet{}, err
	}
	submodules, err := submodulePaths()
	if err != nil {
		return discardSet{}, err
	}
	var cands []newPath
	seen := map[string]bool{}
	add := func(path, kind string) error {
		if path == "" || seen[path] {
			return nil
		}
		seen[path] = true
		c, err := inspectCandidate(root, path, kind, submodules)
		if err != nil {
			return err
		}
		cands = append(cands, c)
		return nil
	}
	for _, path := range unstaged {
		kind := kinds[path]
		if kind == "" {
			kind = "added"
		}
		if err := add(path, kind); err != nil {
			return discardSet{}, err
		}
	}
	headUntracked, err := pathsInHEAD(snap.Untracked)
	if err != nil {
		return discardSet{}, err
	}
	for _, path := range snap.Untracked {
		if headUntracked[path] {
			restore = append(restore, path)
			continue
		}
		if err := add(path, "untracked"); err != nil {
			return discardSet{}, err
		}
	}
	restore = uniquePaths(restore)
	id, err := readIdentity()
	if err != nil {
		return discardSet{}, err
	}
	return discardSet{id: id, cands: cands, unstaged: unstaged, restore: restore}, nil
}

func pathsInHEAD(paths []string) (map[string]bool, error) {
	in := map[string]bool{}
	if len(paths) == 0 {
		return in, nil
	}
	args := append([]string{"--literal-pathspecs", "ls-tree", "-z", "--name-only", "HEAD", "--"}, paths...)
	out, err := gitFromRoot(args...)
	if err != nil {
		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s", msg)
	}
	for _, path := range strings.Split(out, "\x00") {
		if path != "" {
			in[path] = true
		}
	}
	return in, nil
}

func repoRoot() (string, error) {
	out, err := git.Output("rev-parse", "--show-toplevel")
	root := strings.TrimSpace(out)
	if err != nil || root == "" {
		msg := root
		if msg == "" && err != nil {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return filepath.Clean(root), nil
}

func gitFromRoot(args ...string) (string, error) {
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	return git.Output(append([]string{"-C", root}, args...)...)
}

func submodulePaths() (map[string]bool, error) {
	out, err := gitFromRoot("ls-files", "-s", "-z")
	if err != nil {
		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s", msg)
	}
	found := map[string]bool{}
	for _, rec := range strings.Split(out, "\x00") {
		if rec == "" {
			continue
		}
		mode, path, ok := strings.Cut(rec, " ")
		if !ok || mode != "160000" {
			continue
		}
		_, path, ok = strings.Cut(path, "\t")
		if ok && path != "" {
			found[path] = true
		}
	}
	return found, nil
}

func inspectCandidate(root, path, kind string, submodules map[string]bool) (newPath, error) {
	c := newPath{path: path, kind: kind}
	if pathEscapes(path) {
		c.block = "path escapes the repository"
		return c, nil
	}
	if gitMetadata(path) {
		c.block = "git metadata"
		return c, nil
	}
	if submodules[path] {
		c.block = "submodule"
		return c, nil
	}
	if nest, ok := nestedRepo([]string{path}); ok && nest != "" {
		c.block = "nested repository"
		return c, nil
	}
	link, err := symlinkAncestor(root, path)
	if err != nil {
		return newPath{}, err
	}
	if link {
		c.block = "symlink ancestor"
		return c, nil
	}
	fp, err := fingerprint(root, path)
	if err != nil {
		return newPath{}, err
	}
	c.print = fp
	if fp.missing {
		return c, nil
	}
	if fp.mode&os.ModeSymlink != 0 {
		return c, nil
	}
	if fp.mode.IsRegular() {
		return c, nil
	}
	if fp.mode.IsDir() {
		c.block = "directory"
		return c, nil
	}
	c.block = "special file"
	return c, nil
}

func pathEscapes(path string) bool {
	if path == "" || filepath.IsAbs(path) {
		return true
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	return clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.IsAbs(clean)
}

func gitMetadata(path string) bool {
	clean := strings.ReplaceAll(path, `\`, `/`)
	return clean == ".git" || strings.HasPrefix(clean, ".git/") || strings.Contains(clean, "/.git/") || strings.HasSuffix(clean, "/.git")
}

func symlinkAncestor(root, rel string) (bool, error) {
	rel = filepath.Clean(filepath.FromSlash(rel))
	parts := strings.Split(rel, string(filepath.Separator))
	cur := root
	for i := 0; i < len(parts)-1; i++ {
		cur = filepath.Join(cur, parts[i])
		info, err := os.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return true, nil
		}
	}
	return false, nil
}

func fingerprint(root, rel string) (filePrint, error) {
	full := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return filePrint{missing: true}, nil
		}
		return filePrint{}, err
	}
	fp := filePrint{mode: info.Mode(), size: info.Size(), mod: info.ModTime().UnixNano()}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(full)
		if err != nil {
			return filePrint{}, err
		}
		fp.sum = "link:" + target
		return fp, nil
	}
	if !info.Mode().IsRegular() {
		return fp, nil
	}
	f, err := os.Open(full)
	if err != nil {
		return filePrint{}, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return filePrint{}, err
	}
	fp.sum = hex.EncodeToString(h.Sum(nil))
	return fp, nil
}

func (p discardSet) blockedDelete() error {
	var lines []string
	for _, c := range p.cands {
		if c.block == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s (%s)", strconv.Quote(c.path), c.block))
	}
	if len(lines) == 0 {
		return nil
	}
	return workflowText("Delete cannot remove " + strings.Join(lines, ", ") + ". Nothing was changed.")
}

func writeCandidateList(cands []newPath) {
	text.PlainText("New-path files this discard would keep or delete:")
	for _, c := range cands {
		line := "  " + c.kind + ": " + strconv.Quote(c.path)
		if c.block != "" {
			line += " (" + c.block + "; this refuses the delete)"
		}
		text.PlainText(line)
	}
	text.PlainText("Delete removes every listed file permanently and keeps no backup. A protected path refuses the whole delete before anything is changed.")
}

func resolveDisposition(ctx context.Context, verb string, flags dirtyFlags, cands []newPath) (string, error) {
	if len(cands) > 0 {
		writeCandidateList(cands)
	}
	if len(cands) == 0 {
		if flags.untrackedSet {
			return flags.untracked, nil
		}
		return "keep", nil
	}
	if flags.untrackedSet {
		return flags.untracked, nil
	}
	p := prompt.FromContext(ctx)
	if prompt.NonInteractive(p) {
		return "keep", nil
	}
	ans, err := p.Pick("Keep or delete these new-path files", []prompt.Choice{
		{Value: "keep", Description: "Leave every listed file in place."},
		{Value: "delete", Description: "Remove every listed file permanently. There is no backup."},
		{Value: "cancel", Description: "Stop without changing working files."},
	}, "keep")
	if err != nil {
		if errorsIsCancel(err) {
			return "", cancelled(verb)
		}
		return "", err
	}
	switch ans {
	case "keep", "delete":
		return ans, nil
	case "cancel":
		return "", cancelled(verb)
	default:
		return "", fmt.Errorf("invalid choice %q (use keep, delete, or cancel)", ans)
	}
}

func errorsIsCancel(err error) bool {
	return errors.Is(err, prompt.ErrUserCancelled) || errors.Is(err, io.EOF)
}

func bindDisposition(cmd *cobra.Command, ctx context.Context, verb string) error {
	flags, err := readDirtyFlags(cmd)
	if err != nil {
		return err
	}
	before, err := buildDiscardPlan()
	if err != nil {
		return err
	}
	disp, err := resolveDisposition(ctx, verb, flags, before.cands)
	if err != nil {
		return err
	}
	after, err := buildDiscardPlan()
	if err != nil {
		return err
	}
	if !before.same(after) {
		return workflowText(changedBeforePrep)
	}
	if disp == "delete" {
		if err := after.blockedDelete(); err != nil {
			return err
		}
	}
	if cmd != nil {
		if dirty := cmd.Flags().Lookup("dirty"); dirty != nil && strings.TrimSpace(dirty.Value.String()) != "discard" {
			if err := cmd.Flags().Set("dirty", "discard"); err != nil {
				return err
			}
		}
		if cmd.Flags().Lookup("untracked") != nil {
			if err := cmd.Flags().Set("untracked", disp); err != nil {
				return err
			}
		}
	}
	storeAuth(cmd, ctx, discardAuth{plan: after, disp: disp})
	return nil
}

func storeAuth(cmd *cobra.Command, ctx context.Context, auth discardAuth) {
	if cmd == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cmd.SetContext(context.WithValue(ctx, discardAuthKey{}, auth))
}

func authFrom(cmd *cobra.Command) (discardAuth, bool) {
	if cmd == nil || cmd.Context() == nil {
		return discardAuth{}, false
	}
	auth, ok := cmd.Context().Value(discardAuthKey{}).(discardAuth)
	return auth, ok
}

// DiscardRemoves reports an explicit delete disposition.
func DiscardRemoves(cmd *cobra.Command) bool {
	flags, err := readDirtyFlags(cmd)
	return err == nil && flags.untracked == "delete"
}

func leftoverPlan() (discardSet, error) {
	plan, err := buildDiscardPlan()
	if err != nil {
		return discardSet{}, err
	}
	snap, err := state.ReadSnapshot()
	if err != nil {
		return discardSet{}, err
	}
	staged := map[string]bool{}
	for _, path := range snap.Staged {
		staged[path] = true
	}
	var cands []newPath
	for _, c := range plan.cands {
		if !staged[c.path] {
			cands = append(cands, c)
		}
	}
	plan.cands = cands
	return plan, nil
}

func newPathKey() (string, error) {
	plan, err := leftoverPlan()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, c := range plan.cands {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%v\t%d\t%s\n", c.path, c.kind, c.block, c.print.missing, c.print.size, c.print.sum)
	}
	return b.String(), nil
}

// DisposablePaths lists new-path files Delete would remove.
func DisposablePaths() ([]string, error) {
	plan, err := buildDiscardPlan()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, c := range plan.cands {
		if c.block == "" {
			out = append(out, c.path)
		}
	}
	return out, nil
}

func discardThen(cmd *cobra.Command, target string, body func() error) error {
	auth, ok := authFrom(cmd)
	if !ok {
		return fmt.Errorf("discard was not authorized")
	}
	now, err := buildDiscardPlan()
	if err != nil {
		return err
	}
	if !auth.plan.same(now) {
		return workflowText(changedBeforePrep)
	}
	if err := discardCollisions(target, auth.plan, auth.disp); err != nil {
		return err
	}
	var removed []string
	if auth.disp == "delete" {
		var left []string
		removed, left, err = removeCandidates(auth.plan)
		if err != nil {
			return removalStopped(removed, left, false, err)
		}
	}
	if err := revertTrackedEdits(auth.plan.unstaged, auth.plan.restore); err != nil {
		if auth.disp == "delete" {
			return removalStopped(removed, nil, false, err)
		}
		return err
	}
	again, err := state.ReadSnapshot()
	if err != nil {
		if auth.disp == "delete" {
			return removalStopped(removed, nil, true, err)
		}
		return err
	}
	if path, err := blockedPath(target, postOccupants(again, auth.plan, auth.disp), false); err != nil || path != "" {
		if err != nil {
			if auth.disp == "delete" {
				return removalStopped(removed, nil, true, err)
			}
			return err
		}
		msg := fmt.Sprintf("%s would be overwritten. It was left in place. Tracked edits were already discarded, and checkout did not run.", path)
		if auth.disp == "delete" && len(removed) > 0 {
			msg = fmt.Sprintf("%s would be overwritten. It was left in place. Listed new-path files were already removed. %s Checkout did not run.", path, noDiscardBackup)
		}
		return workflowText(msg)
	}
	if err := body(); err != nil {
		if auth.disp == "delete" && len(removed) > 0 {
			text.ErrorText("Those new-path files were already removed permanently. " + noDiscardBackup)
		}
		return err
	}
	return nil
}

func discardCollisions(target string, plan discardSet, disp string) error {
	snap, err := state.ReadSnapshot()
	if err != nil {
		return err
	}
	skip := map[string]bool{}
	for _, path := range plan.restore {
		skip[path] = true
	}
	var occupants []string
	for _, path := range collisionOccupants(snap, plan, disp) {
		if !skip[path] {
			occupants = append(occupants, path)
		}
	}
	if path, err := blockedPath(target, occupants, true); err != nil || path != "" {
		if err != nil {
			return err
		}
		return workflowText(fmt.Sprintf("%s would be overwritten. It was left in place.", path))
	}
	if disp == "delete" {
		return nil
	}
	if path, err := blockedPath(target, plan.unstaged, false); err != nil || path != "" {
		if err != nil {
			return err
		}
		return workflowText(fmt.Sprintf("%s would be overwritten. It was left in place.", path))
	}
	return nil
}

func collisionOccupants(snap state.Snapshot, plan discardSet, disp string) []string {
	drop := map[string]bool{}
	if disp == "delete" {
		for _, c := range plan.cands {
			if c.block == "" {
				drop[c.path] = true
			}
		}
	}
	var out []string
	for _, path := range snap.Untracked {
		if !drop[path] {
			out = append(out, path)
		}
	}
	return append(out, snap.Ignored...)
}

func postOccupants(snap state.Snapshot, plan discardSet, disp string) []string {
	return collisionOccupants(snap, plan, disp)
}

func removeCandidates(plan discardSet) (removed, left []string, err error) {
	var queue []newPath
	for _, c := range plan.cands {
		if c.block == "" && !c.print.missing {
			queue = append(queue, c)
		}
	}
	root, err := repoRoot()
	if err != nil {
		return nil, pathsOf(queue), err
	}
	for i, c := range queue {
		now, ferr := fingerprint(root, c.path)
		if ferr != nil {
			return removed, pathsOf(queue[i:]), ferr
		}
		if now != c.print {
			return removed, pathsOf(queue[i:]), fmt.Errorf("%s changed before it could be removed", strconv.Quote(c.path))
		}
		if rerr := removeWorktreeFile(root, c.path); rerr != nil {
			return removed, pathsOf(queue[i:]), rerr
		}
		removed = append(removed, c.path)
	}
	removeEmptyParents(root, removed)
	return removed, nil, nil
}

func pathsOf(cands []newPath) []string {
	out := make([]string, 0, len(cands))
	for _, c := range cands {
		out = append(out, c.path)
	}
	return out
}

var removeWorktreeFile = removeWithRoot

func removeWithRoot(root, rel string) error {
	r, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer r.Close()
	return r.Remove(filepath.FromSlash(rel))
}

func removeEmptyParents(root string, rels []string) {
	seen := map[string]bool{}
	var dirs []string
	for _, rel := range rels {
		dir := filepath.Dir(filepath.FromSlash(rel))
		for dir != "." && dir != "" {
			if seen[dir] {
				break
			}
			seen[dir] = true
			dirs = append(dirs, dir)
			dir = filepath.Dir(dir)
		}
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return
	}
	defer r.Close()
	sort.Slice(dirs, func(i, j int) bool {
		return strings.Count(dirs[i], string(filepath.Separator)) > strings.Count(dirs[j], string(filepath.Separator))
	})
	for _, dir := range dirs {
		info, err := os.Lstat(filepath.Join(root, dir))
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || holdsWorkingDir(root, dir) {
			continue
		}
		_ = r.Remove(dir)
	}
}

func holdsWorkingDir(root, dir string) bool {
	cwd, err := os.Getwd()
	if err != nil {
		return true
	}
	full := filepath.Join(root, dir)
	cwd, err = filepath.EvalSymlinks(cwd)
	if err != nil {
		return true
	}
	full, err = filepath.EvalSymlinks(full)
	if err != nil {
		return true
	}
	full = filepath.Clean(full)
	cwd = filepath.Clean(cwd)
	rel, err := filepath.Rel(full, cwd)
	if err != nil {
		return true
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func removalStopped(removed, left []string, restored bool, cause error) error {
	var b strings.Builder
	if cause != nil {
		b.WriteString(cause.Error())
		b.WriteString(". ")
	}
	if len(removed) == 0 {
		b.WriteString("No files were removed. ")
	} else {
		b.WriteString("Removed permanently:")
		for _, path := range removed {
			b.WriteString(" ")
			b.WriteString(strconv.Quote(path))
		}
		b.WriteString(". ")
	}
	if len(left) > 0 {
		b.WriteString("Still present:")
		for _, path := range left {
			b.WriteString(" ")
			b.WriteString(strconv.Quote(path))
		}
		b.WriteString(". ")
	}
	if restored {
		b.WriteString("Tracked edits were already restored. ")
	} else {
		b.WriteString("Tracked edits were not restored. ")
	}
	b.WriteString(noDiscardBackup)
	return workflowText(strings.TrimSpace(b.String()))
}

// PolicyOccupants lists paths that block an incoming write for choice.
// Delete does not treat approved new-path files as blockers.
func PolicyOccupants(cmd *cobra.Command, snap state.Snapshot, choice string) ([]string, error) {
	out := append([]string{}, snap.Ignored...)
	if choice != "carry" && choice != "discard" {
		return out, nil
	}
	extra := append([]string{}, snap.Untracked...)
	keep, err := DiscardKeeps()
	if err != nil {
		return nil, err
	}
	extra = append(extra, keep...)
	if choice == "discard" && DiscardRemoves(cmd) {
		drop, err := DisposablePaths()
		if err != nil {
			return nil, err
		}
		extra = withoutPaths(extra, drop)
	}
	return append(out, extra...), nil
}

func withoutPaths(paths, drop []string) []string {
	gone := map[string]bool{}
	for _, path := range drop {
		gone[path] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, path := range paths {
		if path == "" || gone[path] || seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	return out
}
