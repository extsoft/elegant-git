package pipe

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/extsoft/elegant-git/internal/cmdid"
	"github.com/extsoft/elegant-git/internal/git"
	cmdmem "github.com/extsoft/elegant-git/internal/memory/cmd"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/recovery"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
)

type decision struct {
	runBody bool
	opID    string
	legacy  bool
	done    bool
}

type pipeOpt struct {
	restoreOnError   bool
	includeUntracked bool
	retain           bool
	label            string
	// restoreWhenSafe applies a failed body when HEAD is on the recorded
	// destination and no Git operation is in progress. Other failures stay
	// suspended.
	restoreWhenSafe bool
	// beforePrepare runs only when a new stash would be created.
	// It can select retain and the stash label, or refuse before any backup.
	beforePrepare func() (retain bool, label string, err error)
}

func runStashPipe(id cmdid.ID, fn func() error, opt pipeOpt) (Retention, error) {
	if err := cmdmem.ValidateForCurrent(); err != nil {
		return Retention{}, err
	}
	_ = gitQuiet("update-index", "-q", "--really-refresh")

	gitDir, gerr := memrepo.GitDir()
	if gerr != nil {
		return Retention{}, fn()
	}
	common, cerr := memrepo.GitCommonDir()
	if gerr != nil || cerr != nil {
		return Retention{}, runLegacyPipe(id, fn, opt.restoreOnError)
	}

	dec, err := begin(id, gitDir, common, &opt)
	if err != nil {
		return Retention{}, err
	}
	if dec.done {
		return Retention{}, nil
	}
	var bodyErr error
	if dec.opID != "" {
		setCurrentRestore(common, dec.opID)
		defer clearCurrentRestore()
	}
	if dec.runBody {
		bodyErr = fn()
	}
	return finish(id, gitDir, common, dec, bodyErr, opt)
}

func runLegacyPipe(id cmdid.ID, fn func() error, restoreOnError bool) error {
	saved, err := cmdmem.Get(id, cmdmem.FieldStash)
	if err != nil {
		return err
	}
	if saved != "" {
		return legacyReplayError(id.String(), saved)
	}
	if HasChanges() {
		return fmt.Errorf("backup preparation cannot record a recovery journal for this repository")
	}
	return fn()
}

func begin(id cmdid.ID, gitDir, common string, opt *pipeOpt) (decision, error) {
	var dec decision
	err := recovery.WithLock(common, func() error {
		f, err := recovery.Load(common)
		if err != nil {
			return err
		}
		if op, ok, err := f.Own(id.String(), gitDir); err != nil {
			return err
		} else if ok {
			return decideOwn(&dec, common, id, &f, op)
		}
		msg, err := cmdmem.Get(id, cmdmem.FieldStash)
		if err != nil {
			return err
		}
		if msg != "" {
			return legacyReplayError(id.String(), msg)
		}
		if state.InProgressOperation() == "rebase" {
			dec.runBody = true
			return nil
		}
		if opt.includeUntracked {
			snap, err := state.ReadSnapshot()
			if err != nil {
				return err
			}
			if !snap.Relevant() {
				dec.runBody = true
				return nil
			}
		} else if !HasChanges() {
			dec.runBody = true
			return nil
		}
		if others := f.Others(id.String(), gitDir); len(others) > 0 {
			return recovery.PendingError(others[0])
		}
		if err := pendingLegacy(gitDir, id.String()); err != nil {
			return err
		}
		switch state.InProgressOperation() {
		case "merge", "cherry-pick", "revert", "bisect":
			return fmt.Errorf("a %s is in progress; finish or abort it with Git before this command changes files", state.InProgressOperation())
		}
		if opt.beforePrepare != nil {
			retain, label, err := opt.beforePrepare()
			if err != nil {
				return err
			}
			opt.retain = retain
			if label != "" {
				opt.label = label
			}
		}
		return prepare(&dec, id, gitDir, common, &f, *opt)
	})
	return dec, err
}

func decideOwn(dec *decision, common string, id cmdid.ID, f *recovery.File, op recovery.Op) error {
	switch op.Phase {
	case recovery.PhaseApplying:
		return recovery.NoSecondApplyError(op)
	case recovery.PhaseRunning:
		if state.InProgressOperation() == "rebase" {
			if err := allowOwnedRebase(op); err != nil {
				return err
			}
			dec.runBody = true
			dec.opID = op.ID
			return nil
		}
		return recovery.NoReplayError(op)
	case recovery.PhaseSuspended:
		if state.InProgressOperation() == "rebase" {
			if err := allowOwnedRebase(op); err != nil {
				return err
			}
			dec.runBody = true
			dec.opID = op.ID
			return nil
		}
		if cur := state.InProgressOperation(); cur != "" {
			return fmt.Errorf("recovery backup %s was kept because a %s is still in progress. Inspect it with `eg repo doctor`", op.StashOID, cur)
		}
		if !sourceSafe(op) {
			return fmt.Errorf("%s", sourceKeptMessage(op))
		}
		return recovery.NoReplayError(op)
	case recovery.PhaseApplied:
		if err := cleanup(common, id, op, f); err != nil {
			return err
		}
		text.InfoText(fmt.Sprintf("Recovery for %s was already applied. Cleanup finished and the command was not run again.", op.Command))
		dec.done = true
		return nil
	case recovery.PhasePreparing:
		return stopPreparing(common, op, f)
	case recovery.PhasePreserved:
		op.Phase = recovery.PhaseRunning
		f.Upsert(op)
		if err := recovery.Save(common, *f); err != nil {
			return err
		}
		dec.runBody = true
		dec.opID = op.ID
		return nil
	default:
		return fmt.Errorf("unknown recovery phase %q for %s. Inspect it with `eg repo doctor`", op.Phase, op.Command)
	}
}

func stopPreparing(common string, op recovery.Op, f *recovery.File) error {
	if op.StashOID == "" {
		hits, err := listStashes()
		if err != nil {
			return err
		}
		hit, ok, err := uniqueByToken(hits, op.ID)
		if err != nil {
			return err
		}
		if ok {
			op.StashOID = hit.oid
			op.Ref = recovery.RefName(op.ID)
			f.Upsert(op)
			if err := recovery.Save(common, *f); err != nil {
				return err
			}
		}
	}
	if op.StashOID != "" && op.Ref != "" {
		if err := git.Verbose("update-ref", op.Ref, op.StashOID); err != nil {
			return fmt.Errorf("recovery backup %s is recorded but its private ref was not saved: %w", op.StashOID, err)
		}
	}
	return recovery.NoReplayError(op)
}

func prepare(dec *decision, id cmdid.ID, gitDir, common string, f *recovery.File, opt pipeOpt) error {
	if err := recovery.CheckFloor(); err != nil {
		return err
	}
	if err := refuseUnsupported(); err != nil {
		return err
	}
	source, detached, err := captureSource()
	if err != nil {
		return err
	}
	opID, err := recovery.NewID()
	if err != nil {
		return err
	}
	op := recovery.Op{
		ID:       opID,
		Command:  id.String(),
		Worktree: gitDir,
		Source:   source,
		Detached: detached,
		Phase:    recovery.PhasePreparing,
		Message:  stashMessage(id.String(), opID, opt.label),
	}
	f.Upsert(op)
	if err := recovery.Save(common, *f); err != nil {
		return err
	}
	args := []string{"stash", "push"}
	if opt.includeUntracked {
		args = append(args, "-u")
	}
	args = append(args, "--message", op.Message)
	if err := git.Verbose(args...); err != nil {
		return abandonFailedPush(common, f, op, err)
	}
	hits, err := listStashes()
	if err != nil {
		return err
	}
	hit, ok, err := uniqueByToken(hits, op.ID)
	if err != nil {
		return err
	}
	if !ok {
		if !f.Remove(op.ID, "") {
			return fmt.Errorf("recovery stash %s was not found after it was saved; other stashes were not changed. Inspect it with `eg repo doctor`", op.ID)
		}
		if err := recovery.Save(common, *f); err != nil {
			return err
		}
		dec.runBody = true
		return nil
	}
	op.StashOID = hit.oid
	op.Ref = recovery.RefName(op.ID)
	f.Upsert(op)
	if err := recovery.Save(common, *f); err != nil {
		return err
	}
	if err := git.Verbose("update-ref", op.Ref, op.StashOID); err != nil {
		return fmt.Errorf("recovery backup %s is recorded but its private ref was not saved: %w", op.StashOID, err)
	}
	op.Phase = recovery.PhasePreserved
	f.Upsert(op)
	if err := recovery.Save(common, *f); err != nil {
		return err
	}
	op.Phase = recovery.PhaseRunning
	f.Upsert(op)
	if err := recovery.Save(common, *f); err != nil {
		return err
	}
	if err := cmdmem.Set(id, cmdmem.FieldStash, op.Message); err != nil {
		return err
	}
	dec.runBody = true
	dec.opID = op.ID
	return nil
}

// abandonFailedPush drops a preparing record when Git saved nothing.
// A stash that exists despite the error stays recorded for inspection.
func abandonFailedPush(common string, f *recovery.File, op recovery.Op, pushErr error) error {
	hits, err := listStashes()
	if err != nil {
		return combine(pushErr, err)
	}
	_, ok, err := uniqueByToken(hits, op.ID)
	if err != nil {
		return combine(pushErr, err)
	}
	if ok {
		return pushErr
	}
	if !f.Remove(op.ID, "") {
		return pushErr
	}
	if err := recovery.Save(common, *f); err != nil {
		return combine(pushErr, err)
	}
	return pushErr
}

func stashMessage(command, id, label string) string {
	msg := recovery.Message(command, id)
	if label == "" {
		return msg
	}
	return msg + " " + label
}

func finish(id cmdid.ID, gitDir, common string, dec decision, bodyErr error, opt pipeOpt) (Retention, error) {
	if dec.legacy {
		if bodyErr != nil && !opt.restoreOnError {
			return Retention{}, bodyErr
		}
		return Retention{}, combine(bodyErr, popLegacy(id))
	}
	if dec.opID == "" {
		return Retention{}, bodyErr
	}
	var kept Retention
	err := recovery.WithLock(common, func() error {
		f, err := recovery.Load(common)
		if err != nil {
			return combine(bodyErr, err)
		}
		op, ok := f.ByID(dec.opID)
		if !ok {
			return bodyErr
		}
		if bodyErr != nil && !opt.restoreOnError {
			safe, err := restoreAllowed(opt, op)
			if err != nil {
				return combine(bodyErr, err)
			}
			if !safe {
				if op.Phase == recovery.PhaseRunning {
					op.Phase = recovery.PhaseSuspended
					rememberOwnedRebase(&op)
					f.Upsert(op)
					if err := recovery.Save(common, f); err != nil {
						return combine(bodyErr, err)
					}
				}
				if op.StashOID != "" {
					kept = Retention{OID: op.StashOID}
				}
				return bodyErr
			}
		}
		if cur := state.InProgressOperation(); cur != "" {
			return combine(bodyErr, fmt.Errorf("recovery backup %s was kept because a %s is still in progress. Inspect it with `eg repo doctor`", op.StashOID, cur))
		}
		if opt.retain && bodyErr == nil {
			got, err := handoffRetain(common, id, op, &f, opt.label)
			kept = got
			return err
		}
		if !sourceSafe(op) {
			msg := sourceKeptMessage(op)
			text.InfoText(msg)
			if op.Phase == recovery.PhaseRunning {
				op.Phase = recovery.PhaseSuspended
				f.Upsert(op)
				if err := recovery.Save(common, f); err != nil {
					return combine(bodyErr, err)
				}
			}
			if bodyErr != nil {
				return combine(bodyErr, fmt.Errorf("%s", msg))
			}
			return nil
		}
		if ok, err := onDestination(op); err != nil {
			return combine(bodyErr, err)
		} else if !ok {
			return keepOffDestination(common, bodyErr, &op, &f)
		}
		if op.Phase == recovery.PhaseApplying {
			return combine(bodyErr, recovery.NoSecondApplyError(op))
		}
		if op.Phase == recovery.PhaseApplied {
			return combine(bodyErr, cleanup(common, id, op, &f))
		}
		op.Phase = recovery.PhaseApplying
		f.Upsert(op)
		if err := recovery.Save(common, f); err != nil {
			return combine(bodyErr, err)
		}
		if beforeRestoreApply != nil {
			beforeRestoreApply()
		}
		if ok, err := onDestination(op); err != nil {
			return combine(bodyErr, err)
		} else if !ok {
			return combine(bodyErr, fmt.Errorf("%s It will not be applied again.", destinationKeptMessage(op)))
		}
		if err := git.Verbose("stash", "apply", "--index", op.StashOID); err != nil {
			return combine(bodyErr, fmt.Errorf("recovery apply failed; backup %s was kept. The destination may already contain part of that backup: %w", op.StashOID, err))
		}
		if afterRestoreApply != nil {
			afterRestoreApply()
		}
		if ok, err := onDestination(op); err != nil {
			return combine(bodyErr, err)
		} else if !ok {
			return combine(bodyErr, landedOffDestination(op))
		}
		op.Phase = recovery.PhaseApplied
		f.Upsert(op)
		if err := recovery.Save(common, f); err != nil {
			return combine(bodyErr, err)
		}
		return combine(bodyErr, cleanup(common, id, op, &f))
	})
	return kept, err
}

// BeforeRetainHandoff runs after a navigation command succeeds and before that
// stash is released as an ordinary stash. Tests use it to break the handoff.
var BeforeRetainHandoff func()

func handoffRetain(common string, id cmdid.ID, op recovery.Op, f *recovery.File, label string) (Retention, error) {
	if BeforeRetainHandoff != nil {
		BeforeRetainHandoff()
	}
	hits, err := listStashes()
	if err != nil {
		return Retention{}, err
	}
	var match []stashHit
	for _, hit := range hits {
		if hit.oid == op.StashOID && strings.Contains(hit.subject, op.ID) {
			match = append(match, hit)
		}
	}
	if len(match) != 1 {
		return Retention{}, fmt.Errorf("stash handoff for %s did not stay reachable. The backup and recovery record were kept. Inspect it with `eg repo doctor`", op.StashOID)
	}
	if err := deleteRecoveryRef(op); err != nil {
		return Retention{}, err
	}
	if err := clearStashField(id, op.Message); err != nil {
		return Retention{}, err
	}
	if !f.Remove(op.ID, op.StashOID) {
		return Retention{}, fmt.Errorf("recovery %s is not in the journal", op.ID)
	}
	if err := recovery.Save(common, *f); err != nil {
		return Retention{}, err
	}
	return Retention{Label: label, OID: op.StashOID, Selector: match[0].ref}, nil
}

// beforeRecoveryCleanup runs before cleanup resolves the owned stash.
// popLegacy runs it after that legacy lookup and before it refuses to pop.
// Tests change the stash list in that window.
var beforeRecoveryCleanup func()

func cleanup(common string, id cmdid.ID, op recovery.Op, f *recovery.File) error {
	if beforeRecoveryCleanup != nil {
		beforeRecoveryCleanup()
	}
	if err := git.DropStash(op.StashOID, op.Message); err != nil {
		return cleanupFailed(op.StashOID, err)
	}
	if err := deleteRecoveryRef(op); err != nil {
		return cleanupFailed(op.StashOID, err)
	}
	if err := clearStashField(id, op.Message); err != nil {
		return cleanupFailed(op.StashOID, err)
	}
	if !f.Remove(op.ID, op.StashOID) {
		return cleanupFailed(op.StashOID, fmt.Errorf("recovery %s is not in the journal", op.ID))
	}
	if err := recovery.Save(common, *f); err != nil {
		return cleanupFailed(op.StashOID, err)
	}
	return nil
}

func cleanupFailed(oid string, err error) error {
	return fmt.Errorf("Changes were restored; temporary stash cleanup failed for %s: %w", oid, err)
}

func legacyReplayError(command, message string) error {
	refs, err := legacyRefs(message)
	if err != nil {
		return err
	}
	if len(refs) == 0 {
		return fmt.Errorf("recovery stash for %s was not found; other stashes were not changed. Inspect it with `eg repo doctor`. This command will not run again automatically", command)
	}
	if len(refs) > 1 {
		return fmt.Errorf("recovery stash for %s is ambiguous; other stashes were not changed. Inspect it with `eg repo doctor`. This command will not run again automatically", command)
	}
	return fmt.Errorf("pending recovery stash for %s is still recorded. Inspect it with `eg repo doctor`. This command will not run again automatically", command)
}

func deleteRecoveryRef(op recovery.Op) error {
	if op.Ref == "" || op.StashOID == "" {
		return nil
	}
	if _, err := git.Output("rev-parse", "--verify", "--quiet", op.Ref); err != nil {
		return nil
	}
	return git.Verbose("update-ref", "-d", op.Ref, op.StashOID)
}

func clearStashField(id cmdid.ID, message string) error {
	msg, err := cmdmem.Get(id, cmdmem.FieldStash)
	if err != nil {
		return err
	}
	if msg == "" || (message != "" && msg != message) {
		return nil
	}
	return cmdmem.Unset(id, cmdmem.FieldStash)
}

func pendingLegacy(gitDir, command string) error {
	st, err := cmdmem.Load(gitDir)
	if err != nil {
		return err
	}
	for key, entry := range st {
		if key == command || entry == nil || entry.Stash == "" {
			continue
		}
		return fmt.Errorf("pending recovery stash for %s is still recorded. Inspect it with `eg repo doctor`. This command will not change that backup", key)
	}
	return nil
}

// beforeRestoreApply runs after the applying phase is saved and before the
// last destination check. Tests move HEAD in that window.
var beforeRestoreApply func()

// afterRestoreApply runs after git stash apply returns and before HEAD is
// read again. Tests move HEAD in that window.
var afterRestoreApply func()

type restoreOwner struct {
	common string
	id     string
}

var currentRestore restoreOwner

func setCurrentRestore(common, id string) {
	currentRestore = restoreOwner{common: common, id: id}
}

func clearCurrentRestore() {
	currentRestore = restoreOwner{}
}

// RememberRestoreBranch records the branch a navigation command checked out.
// Restore then requires that branch, not merely that the original branch exists.
func RememberRestoreBranch(branch string) error {
	if currentRestore.id == "" || strings.TrimSpace(branch) == "" {
		return nil
	}
	return recovery.WithLock(currentRestore.common, func() error {
		f, err := recovery.Load(currentRestore.common)
		if err != nil {
			return err
		}
		op, ok := f.ByID(currentRestore.id)
		if !ok {
			return fmt.Errorf("recovery %s is not in the journal", currentRestore.id)
		}
		op.RestoreBranch = branch
		f.Upsert(op)
		return recovery.Save(currentRestore.common, f)
	})
}

func keepOffDestination(common string, bodyErr error, op *recovery.Op, f *recovery.File) error {
	msg := destinationKeptMessage(*op)
	if op.Phase == recovery.PhaseRunning {
		op.Phase = recovery.PhaseSuspended
		f.Upsert(*op)
		if err := recovery.Save(common, *f); err != nil {
			return combine(bodyErr, err)
		}
	}
	return combine(bodyErr, fmt.Errorf("%s", msg))
}

func destinationKeptMessage(op recovery.Op) string {
	if gitDir, err := memrepo.GitDir(); err != nil || !sameGitPath(op.Worktree, gitDir) {
		return fmt.Sprintf("Uncommitted changes stay in the stash because this is not the worktree that saved backup %s. Inspect it with `eg repo doctor`. This command will not run again automatically.", op.StashOID)
	}
	if op.RestoreBranch != "" {
		return fmt.Sprintf("Uncommitted changes stay in the stash because HEAD is %s, not %s. Backup %s was kept. Inspect it with `eg repo doctor`. This command will not run again automatically.", headPlace(), approvedPlace(op), op.StashOID)
	}
	if op.Detached {
		return fmt.Sprintf("Uncommitted changes stay in the stash because HEAD is no longer detached at %s. Backup %s was kept. Inspect it with `eg repo doctor`. This command will not run again automatically.", op.Source, op.StashOID)
	}
	return fmt.Sprintf("Uncommitted changes stay in the stash because HEAD is %s, not %s. Backup %s was kept. Inspect it with `eg repo doctor`. This command will not run again automatically.", headPlace(), approvedPlace(op), op.StashOID)
}

func landedOffDestination(op recovery.Op) error {
	return fmt.Errorf("Changes from backup %s landed on %s, not %s. The stash entry remains. The journal and private ref were kept. Inspect it with `eg repo doctor`. This command will not apply it again.", op.StashOID, headPlace(), approvedPlace(op))
}

func approvedPlace(op recovery.Op) string {
	if op.RestoreBranch != "" {
		return op.RestoreBranch
	}
	if op.Detached {
		return "detached at " + shortOID(op.Source)
	}
	if op.Source == "" {
		return "unknown"
	}
	return op.Source
}

func headPlace() string {
	head, err := headBranch()
	if err != nil || head == "" {
		return "unknown"
	}
	if head != "HEAD" {
		return head
	}
	oid, err := git.Output("rev-parse", "--short", "HEAD")
	if err != nil || strings.TrimSpace(oid) == "" {
		return "detached"
	}
	return "detached at " + strings.TrimSpace(oid)
}

func shortOID(oid string) string {
	oid = strings.TrimSpace(oid)
	if len(oid) > 12 {
		return oid[:12]
	}
	return oid
}

func detachedAt(commit string) bool {
	ref, err := git.Output("symbolic-ref", "--quiet", "HEAD")
	if err == nil && strings.TrimSpace(ref) != "" {
		return false
	}
	oid, err := git.Output("rev-parse", "--verify", "HEAD")
	return err == nil && strings.TrimSpace(oid) == commit
}

func onDestination(op recovery.Op) (bool, error) {
	gitDir, err := memrepo.GitDir()
	if err != nil {
		return false, err
	}
	if !sameGitPath(op.Worktree, gitDir) {
		return false, nil
	}
	if op.RestoreBranch != "" {
		head, err := headBranch()
		if err != nil {
			return false, err
		}
		return head == op.RestoreBranch, nil
	}
	if op.Detached {
		return detachedAt(op.Source), nil
	}
	head, err := headBranch()
	if err != nil {
		return false, err
	}
	return head == op.Source, nil
}

func headBranch() (string, error) {
	out, err := git.Output("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func sameGitPath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	ra, ea := filepath.EvalSymlinks(a)
	rb, eb := filepath.EvalSymlinks(b)
	return ea == nil && eb == nil && ra == rb
}

func restoreAllowed(opt pipeOpt, op recovery.Op) (bool, error) {
	if opt.retain || !opt.restoreWhenSafe || state.InProgressOperation() != "" || !sourceSafe(op) {
		return false, nil
	}
	return onDestination(op)
}

func sourceSafe(op recovery.Op) bool {
	if op.RestoreBranch != "" {
		return state.LocalBranchExists(op.RestoreBranch)
	}
	if op.Detached {
		ref, err := git.Output("symbolic-ref", "--quiet", "HEAD")
		if err == nil && strings.TrimSpace(ref) != "" {
			return false
		}
		oid, err := git.Output("rev-parse", "--verify", "HEAD")
		return err == nil && strings.TrimSpace(oid) == op.Source
	}
	if op.Source == "" {
		return false
	}
	return state.LocalBranchExists(op.Source)
}

func sourceKeptMessage(op recovery.Op) string {
	if op.RestoreBranch != "" {
		return fmt.Sprintf("Uncommitted changes stay in the stash because the '%s' branch no longer exists. Backup %s was kept. Inspect it with `eg repo doctor`.", op.RestoreBranch, op.StashOID)
	}
	if op.Detached {
		return fmt.Sprintf("Uncommitted changes stay in the stash because HEAD is no longer detached at %s. Backup %s was kept. Inspect it with `eg repo doctor`.", op.Source, op.StashOID)
	}
	return fmt.Sprintf("Uncommitted changes stay in the stash because the '%s' branch no longer exists. Backup %s was kept. Inspect it with `eg repo doctor`.", op.Source, op.StashOID)
}

func rememberOwnedRebase(op *recovery.Op) {
	if op.RebaseHead != "" || op.RebaseOnto != "" || op.RebaseOrig != "" {
		return
	}
	if state.InProgressOperation() != "rebase" {
		return
	}
	cur, ok := state.ActiveRebase()
	if !ok {
		return
	}
	op.RebaseHead = cur.Head
	op.RebaseOnto = cur.Onto
	op.RebaseOrig = cur.Orig
}

func allowOwnedRebase(op recovery.Op) error {
	cur, ok := state.ActiveRebase()
	owned := op.RebaseHead != "" && op.RebaseOnto != "" && op.RebaseOrig != ""
	if !ok || !owned || cur.Head != op.RebaseHead || cur.Onto != op.RebaseOnto || cur.Orig != op.RebaseOrig {
		return fmt.Errorf("recovery backup %s was kept because the active rebase is not owned by this backup. Inspect it with `eg repo doctor`. This command will not continue it or restore the backup", op.StashOID)
	}
	return nil
}

func captureSource() (string, bool, error) {
	ref, err := git.Output("symbolic-ref", "--quiet", "HEAD")
	ref = strings.TrimSpace(ref)
	if err == nil && ref != "" {
		return strings.TrimPrefix(ref, "refs/heads/"), false, nil
	}
	oid, err := git.Output("rev-parse", "--verify", "HEAD")
	oid = strings.TrimSpace(oid)
	if err != nil || oid == "" {
		return "", false, fmt.Errorf("backup preparation cannot identify HEAD")
	}
	return oid, true, nil
}

func refuseUnsupported() error {
	if state.IsBareRepository() {
		return fmt.Errorf("backup preparation is not supported in a bare repository")
	}
	if state.HeadUnborn() {
		return fmt.Errorf("backup preparation is not supported before the first commit")
	}
	if out, err := git.Output("config", "--local", "--get", "core.sparseCheckout"); err == nil && strings.EqualFold(strings.TrimSpace(out), "true") {
		return fmt.Errorf("backup preparation does not support a sparse checkout")
	}
	if out, err := git.Output("config", "--local", "--get", "core.splitIndex"); err == nil && strings.EqualFold(strings.TrimSpace(out), "true") {
		return fmt.Errorf("backup preparation does not support a split index")
	}
	unmerged, err := git.Output("ls-files", "-u")
	if err != nil {
		return err
	}
	if strings.TrimSpace(unmerged) != "" && state.InProgressOperation() == "" {
		return fmt.Errorf("backup preparation does not support unmerged index entries")
	}
	flagged, err := git.Output("ls-files", "-v")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(flagged, "\n") {
		if line == "" {
			continue
		}
		switch line[0] {
		case 'S', 's':
			return fmt.Errorf("backup preparation does not support skip-worktree path %s", strings.TrimSpace(line[1:]))
		}
	}
	st, err := git.Output("status", "--porcelain=v2", "-z", "--untracked-files=no")
	if err != nil {
		return err
	}
	for _, rec := range strings.Split(st, "\x00") {
		fields := strings.Fields(rec)
		if len(fields) >= 2 && fields[0] == "1" && strings.HasPrefix(fields[1], ".A") {
			return fmt.Errorf("backup preparation does not support intent-to-add paths")
		}
	}
	sub, err := git.Output("submodule", "status")
	if err != nil {
		if !strings.Contains(sub, "no submodule mapping found") {
			return fmt.Errorf("backup preparation cannot read submodule state: %w", err)
		}
		sub = ""
	}
	for _, line := range strings.Split(sub, "\n") {
		if line == "" {
			continue
		}
		switch line[0] {
		case '+', 'U':
			return fmt.Errorf("backup preparation does not support a dirty submodule")
		}
	}
	return nil
}

type stashHit struct {
	oid     string
	ref     string
	subject string
}

func listStashes() ([]stashHit, error) {
	out, err := git.Output("stash", "list", "--format=%H%x00%gd%x00%s")
	if err != nil {
		return nil, err
	}
	var hits []stashHit
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\x00")
		if len(parts) < 3 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("recovery cannot read the stash list")
		}
		hits = append(hits, stashHit{oid: parts[0], ref: parts[1], subject: parts[2]})
	}
	return hits, nil
}

func uniqueByToken(hits []stashHit, token string) (stashHit, bool, error) {
	var match []stashHit
	for _, hit := range hits {
		if strings.Contains(hit.subject, token) {
			match = append(match, hit)
		}
	}
	switch len(match) {
	case 0:
		return stashHit{}, false, nil
	case 1:
		return match[0], true, nil
	default:
		return stashHit{}, false, fmt.Errorf("recovery stash %s is ambiguous; other stashes were not changed", token)
	}
}

func popLegacy(id cmdid.ID) error {
	saved, err := cmdmem.Get(id, cmdmem.FieldStash)
	if err != nil {
		return err
	}
	if saved == "" {
		return nil
	}
	_ = gitQuiet("update-index", "-q", "--really-refresh")
	refs, err := legacyRefs(saved)
	if err != nil {
		return err
	}
	if len(refs) == 0 {
		return fmt.Errorf("recovery stash for %s was not found; other stashes were not changed. Inspect it with `eg repo doctor`", id.String())
	}
	if len(refs) > 1 {
		return fmt.Errorf("recovery stash for %s is ambiguous; other stashes were not changed. Inspect it with `eg repo doctor`", id.String())
	}
	if source := stashWIPBranch(saved); source != "" && !state.LocalBranchExists(source) {
		text.InfoText(fmt.Sprintf("Uncommitted changes stay in the stash because the '%s' branch no longer exists. Inspect it with `eg repo doctor`.", source))
		return nil
	}
	if beforeRecoveryCleanup != nil {
		beforeRecoveryCleanup()
	}
	return fmt.Errorf("recovery stash for %s is still recorded. Elegant Git will not pop it, because Git cannot remove that one entry without possibly removing a different stash. Inspect it with `eg repo doctor`. This command will not run again automatically", id.String())
}

func legacyRefs(message string) ([]string, error) {
	out, err := git.Output("stash", "list", "--grep="+message, "--format=%gd")
	if err != nil {
		return nil, err
	}
	var refs []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			refs = append(refs, line)
		}
	}
	return refs, nil
}

func combine(primary, secondary error) error {
	if secondary == nil {
		return primary
	}
	if primary == nil {
		return secondary
	}
	return fmt.Errorf("%w\nalso: %v", primary, secondary)
}
