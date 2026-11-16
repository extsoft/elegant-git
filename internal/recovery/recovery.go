// Package recovery records reachable backups for commands that stash work.
// The journal lives in the common git directory so linked worktrees share one owner.
package recovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/extsoft/elegant-git/internal/git"
	cmdmem "github.com/extsoft/elegant-git/internal/memory/cmd"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/uuidv7"
)

const RefPrefix = "refs/elegant-git/recovery/"

// Phase is a journaled step. An applying record is never applied again automatically.
type Phase string

const (
	PhasePreparing Phase = "preparing"
	PhasePreserved Phase = "preserved"
	PhaseRunning   Phase = "running"
	PhaseSuspended Phase = "suspended"
	PhaseApplying  Phase = "applying"
	PhaseApplied   Phase = "applied"
)

// Op is one backup owned by a command in one worktree.
type Op struct {
	ID       string `json:"id"`
	Command  string `json:"command"`
	Worktree string `json:"worktree"`
	Source   string `json:"source"`
	Detached bool   `json:"detached,omitempty"`
	Phase    Phase  `json:"phase"`
	StashOID string `json:"stashOid,omitempty"`
	Ref      string `json:"ref,omitempty"`
	Message  string `json:"message,omitempty"`
}

// File is the common-repo recovery journal.
type File struct {
	Ops          []Op     `json:"ops,omitempty"`
	Acknowledged []string `json:"acknowledged,omitempty"`
}

// NewID is the operation id source. Tests replace it.
var NewID = uuidv7.New

// Message is the stash subject. The id is the identity, not the timestamp.
func Message(command, id string) string {
	return fmt.Sprintf("eg %s auto-stash %s", command, id)
}

// RefName is the private ref that keeps the backup reachable.
func RefName(id string) string {
	return RefPrefix + id
}

// StashMessage strips Git's "On <branch>: " prefix from a stash subject.
func StashMessage(subject string) string {
	if rest, ok := strings.CutPrefix(subject, "On "); ok {
		if _, msg, ok := strings.Cut(rest, ": "); ok {
			return msg
		}
	}
	return subject
}

// IsRecoverySubject reports stash subjects created by current or legacy pipes.
// git stash push --message stores that message after "On <branch>: ".
func IsRecoverySubject(subject string) bool {
	msg := StashMessage(subject)
	return strings.HasPrefix(msg, "eg ") && strings.Contains(msg, " auto-stash")
}

func journalPath(common string) string {
	return filepath.Join(common, "elegant-git", "recovery.json")
}

// WithLock serializes recovery journal updates for one common git directory.
func WithLock(common string, fn func() error) error {
	dir := filepath.Join(common, "elegant-git")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, "recovery.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := lockExclusive(f); err != nil {
		return err
	}
	defer unlockExclusive(f)
	return fn()
}

// Load reads the journal. A missing file is an empty journal.
func Load(common string) (File, error) {
	data, err := os.ReadFile(journalPath(common))
	if err != nil {
		if os.IsNotExist(err) {
			return File{}, nil
		}
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return File{}, fmt.Errorf("recovery journal is unreadable: %w", err)
	}
	return f, nil
}

// Save atomically writes the journal and syncs it before rename.
func Save(common string, f File) error {
	dir := filepath.Join(common, "elegant-git")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	p := journalPath(common)
	tmp := p + ".tmp"
	fh, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := fh.Write(data); err != nil {
		_ = fh.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := fh.Sync(); err != nil {
		_ = fh.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := fh.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// Own returns the single operation for this command and worktree.
func (f File) Own(command, worktree string) (Op, bool, error) {
	var found *Op
	for i := range f.Ops {
		if f.Ops[i].Command != command || !sameWorktree(f.Ops[i].Worktree, worktree) {
			continue
		}
		if found != nil {
			return Op{}, false, fmt.Errorf("ambiguous recovery for %s in %s", command, worktree)
		}
		op := f.Ops[i]
		found = &op
	}
	if found == nil {
		return Op{}, false, nil
	}
	return *found, true, nil
}

// Others returns pending operations that this command in this worktree does not own.
func (f File) Others(command, worktree string) []Op {
	var out []Op
	for _, op := range f.Ops {
		if op.Command == command && sameWorktree(op.Worktree, worktree) {
			continue
		}
		out = append(out, op)
	}
	return out
}

// ByID returns the operation with id.
func (f File) ByID(id string) (Op, bool) {
	for _, op := range f.Ops {
		if op.ID == id {
			return op, true
		}
	}
	return Op{}, false
}

// Remove drops an operation and records its oid as acknowledged.
// It does not drop the stash or delete the private ref.
func (f *File) Remove(id, oid string) bool {
	next := make([]Op, 0, len(f.Ops))
	found := false
	for _, op := range f.Ops {
		if op.ID == id {
			found = true
			if oid == "" {
				oid = op.StashOID
			}
			continue
		}
		next = append(next, op)
	}
	if !found {
		return false
	}
	f.Ops = next
	if oid != "" && !acked(f.Acknowledged, oid) {
		f.Acknowledged = append(f.Acknowledged, oid)
	}
	return true
}

// Upsert replaces the operation with the same id or appends it.
func (f *File) Upsert(op Op) {
	for i := range f.Ops {
		if f.Ops[i].ID == op.ID {
			f.Ops[i] = op
			return
		}
	}
	f.Ops = append(f.Ops, op)
}

// Forget removes an operation and records its oid so prune can offer that stash later.
// It does not drop the stash or delete the private ref.
func Forget(common, id, oid string) error {
	return WithLock(common, func() error {
		f, err := Load(common)
		if err != nil {
			return err
		}
		next := make([]Op, 0, len(f.Ops))
		var found bool
		for _, op := range f.Ops {
			if op.ID == id {
				found = true
				if oid == "" {
					oid = op.StashOID
				}
				continue
			}
			next = append(next, op)
		}
		if !found {
			return fmt.Errorf("recovery %s is not in the journal", id)
		}
		f.Ops = next
		if oid != "" && !acked(f.Acknowledged, oid) {
			f.Acknowledged = append(f.Acknowledged, oid)
		}
		return Save(common, f)
	})
}

func acked(ids []string, oid string) bool {
	for _, id := range ids {
		if id == oid {
			return true
		}
	}
	return false
}

// PendingError tells the user how to inspect someone else's backup.
func PendingError(op Op) error {
	return fmt.Errorf("%s", describe(op, "This command will not change that backup"))
}

// NoReplayError refuses to run a command body whose outcome is unknown.
func NoReplayError(op Op) error {
	return fmt.Errorf("%s", describe(op, "This command will not run again automatically"))
}

// NoSecondApplyError refuses another automatic apply.
func NoSecondApplyError(op Op) error {
	return fmt.Errorf("%s", describe(op, "This backup will not be applied again automatically"))
}

func describe(op Op, action string) string {
	source := op.Source
	if op.Detached {
		source = "detached " + op.Source
	}
	if source == "" {
		source = "unknown source"
	}
	backup := op.StashOID
	if backup == "" {
		backup = "not identified yet"
	}
	ref := op.Ref
	if ref == "" {
		ref = "no private ref yet"
	}
	return fmt.Sprintf("pending recovery for %s (phase %s, source %s, backup %s, ref %s). Inspect it with `eg repo doctor`. %s",
		op.Command, op.Phase, source, backup, ref, action)
}

// Guard blocks commands that do not own a pending backup.
// exempt is set for inspection and for prune, which filters backups itself.
// An unsafe command-state override is rejected even when exempt.
func Guard(command string, exempt bool) error {
	if err := cmdmem.ValidateForCurrent(); err != nil {
		return err
	}
	if exempt {
		return nil
	}
	gitDir, err := memrepo.GitDir()
	if err != nil {
		return nil
	}
	common, err := memrepo.GitCommonDir()
	if err != nil {
		return nil
	}
	f, err := Load(common)
	if err != nil {
		return err
	}
	if others := f.Others(command, gitDir); len(others) > 0 {
		return PendingError(others[0])
	}
	st, err := cmdmem.Load(gitDir)
	if err != nil {
		return err
	}
	for id, entry := range st {
		if entry == nil || entry.Stash == "" || id == command {
			continue
		}
		if _, ok, err := f.Own(id, gitDir); err != nil {
			return err
		} else if ok {
			continue
		}
		return fmt.Errorf("pending recovery stash for %s is still recorded. Inspect it with `eg repo doctor`. This command will not change that backup", id)
	}
	return nil
}

// Protects reports whether a stash must stay out of prune.
// Journaled backups, private refs, and stash messages still recorded in command
// memory are protected. A leftover pipe subject with no record is not.
func Protects(common, oid, subject string) (bool, error) {
	msg := StashMessage(subject)
	if common != "" {
		f, err := Load(common)
		if err != nil {
			return true, err
		}
		for _, op := range f.Ops {
			if oid != "" && op.StashOID == oid {
				return true, nil
			}
			if op.Message != "" && msg == op.Message {
				return true, nil
			}
		}
		if oid != "" {
			owned, err := refOwns(oid)
			if err != nil {
				return true, err
			}
			if owned {
				return true, nil
			}
		}
		recorded, err := recordedStashMessages(common)
		if err != nil {
			return true, err
		}
		if msg != "" && recorded[msg] {
			return true, nil
		}
		return false, nil
	}
	if oid != "" {
		owned, err := refOwns(oid)
		if err != nil {
			return true, err
		}
		if owned {
			return true, nil
		}
	}
	return false, nil
}

func recordedStashMessages(common string) (map[string]bool, error) {
	dirs := []string{common}
	entries, err := os.ReadDir(filepath.Join(common, "worktrees"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(common, "worktrees", entry.Name()))
		}
	}
	out := map[string]bool{}
	for _, dir := range dirs {
		msgs, err := stashMessagesAt(filepath.Join(dir, "elegant-git", "commands.json"))
		if err != nil {
			return nil, err
		}
		for _, msg := range msgs {
			out[msg] = true
		}
	}
	return out, nil
}

func stashMessagesAt(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var st map[string]struct {
		Stash string `json:"stash"`
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("command state %s is unreadable: %w", path, err)
	}
	var msgs []string
	for _, entry := range st {
		if entry.Stash != "" {
			msgs = append(msgs, entry.Stash)
		}
	}
	return msgs, nil
}

func sameWorktree(a, b string) bool {
	if a == b {
		return true
	}
	ra, ea := filepath.EvalSymlinks(a)
	rb, eb := filepath.EvalSymlinks(b)
	return ea == nil && eb == nil && ra == rb
}

func refOwns(oid string) (bool, error) {
	if oid == "" {
		return false, nil
	}
	out, err := git.Output("for-each-ref", "--format=%(objectname)", RefPrefix)
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == oid {
			return true, nil
		}
	}
	return false, nil
}

// RecoveryRefs lists private recovery refs and their oids.
func RecoveryRefs() ([]Ref, error) {
	out, err := git.Output("for-each-ref", "--format=%(objectname)%00%(refname)", RefPrefix)
	if err != nil {
		return nil, err
	}
	var refs []Ref
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		oid, name, ok := strings.Cut(line, "\x00")
		if !ok || oid == "" || name == "" {
			continue
		}
		refs = append(refs, Ref{OID: oid, Name: name})
	}
	return refs, nil
}

// Ref is a private recovery ref.
type Ref struct {
	OID  string
	Name string
}
