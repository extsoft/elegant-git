package pipe

import (
	"fmt"
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

func runStashPipe(id cmdid.ID, fn func() error, restoreOnError bool) error {
	if err := cmdmem.ValidateForCurrent(); err != nil {
		return err
	}
	_ = gitQuiet("update-index", "-q", "--really-refresh")

	gitDir, gerr := memrepo.GitDir()
	if gerr != nil {
		return fn()
	}
	common, cerr := memrepo.GitCommonDir()
	if gerr != nil || cerr != nil {
		return runLegacyPipe(id, fn, restoreOnError)
	}

	dec, err := begin(id, gitDir, common)
	if err != nil {
		return err
	}
	if dec.done {
		return nil
	}
	var bodyErr error
	if dec.runBody {
		bodyErr = fn()
	}
	return finish(id, gitDir, common, dec, bodyErr, restoreOnError)
}

func runLegacyPipe(id cmdid.ID, fn func() error, restoreOnError bool) error {
	saved, err := cmdmem.Get(id, cmdmem.FieldStash)
	if err != nil {
		return err
	}
	if saved == "" && HasChanges() {
		return fmt.Errorf("backup preparation cannot record a recovery journal for this repository")
	}
	bodyErr := fn()
	if bodyErr != nil && !restoreOnError {
		return bodyErr
	}
	if saved == "" {
		return bodyErr
	}
	return combine(bodyErr, popLegacy(id))
}

func begin(id cmdid.ID, gitDir, common string) (decision, error) {
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
			dec.legacy = true
			dec.runBody = true
			return nil
		}
		if state.InProgressOperation() == "rebase" {
			dec.runBody = true
			return nil
		}
		if !HasChanges() {
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
		return prepare(&dec, id, gitDir, common, &f)
	})
	return dec, err
}

func decideOwn(dec *decision, common string, id cmdid.ID, f *recovery.File, op recovery.Op) error {
	switch op.Phase {
	case recovery.PhaseApplying:
		return recovery.NoSecondApplyError(op)
	case recovery.PhaseRunning:
		if state.InProgressOperation() == "rebase" {
			dec.runBody = true
			dec.opID = op.ID
			return nil
		}
		return recovery.NoReplayError(op)
	case recovery.PhaseSuspended:
		if !sourceSafe(op) {
			return fmt.Errorf("%s", sourceKeptMessage(op))
		}
		dec.runBody = true
		dec.opID = op.ID
		return nil
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

func prepare(dec *decision, id cmdid.ID, gitDir, common string, f *recovery.File) error {
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
		Message:  recovery.Message(id.String(), opID),
	}
	f.Upsert(op)
	if err := recovery.Save(common, *f); err != nil {
		return err
	}
	if err := git.Verbose("stash", "push", "--message", op.Message); err != nil {
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

func finish(id cmdid.ID, gitDir, common string, dec decision, bodyErr error, restoreOnError bool) error {
	if dec.legacy {
		if bodyErr != nil && !restoreOnError {
			return bodyErr
		}
		return combine(bodyErr, popLegacy(id))
	}
	if dec.opID == "" {
		return bodyErr
	}
	return recovery.WithLock(common, func() error {
		f, err := recovery.Load(common)
		if err != nil {
			return combine(bodyErr, err)
		}
		op, ok := f.ByID(dec.opID)
		if !ok {
			return bodyErr
		}
		if bodyErr != nil && !restoreOnError {
			if op.Phase == recovery.PhaseRunning {
				op.Phase = recovery.PhaseSuspended
				f.Upsert(op)
				if err := recovery.Save(common, f); err != nil {
					return combine(bodyErr, err)
				}
			}
			return bodyErr
		}
		if cur := state.InProgressOperation(); cur != "" {
			return combine(bodyErr, fmt.Errorf("recovery backup %s was kept because a %s is still in progress. Inspect it with `eg repo doctor`", op.StashOID, cur))
		}
		if !sourceSafe(op) {
			text.InfoText(sourceKeptMessage(op))
			if op.Phase == recovery.PhaseRunning {
				op.Phase = recovery.PhaseSuspended
				f.Upsert(op)
				if err := recovery.Save(common, f); err != nil {
					return combine(bodyErr, err)
				}
			}
			return bodyErr
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
		if err := git.Verbose("stash", "apply", "--index", op.StashOID); err != nil {
			return combine(bodyErr, fmt.Errorf("recovery apply failed; backup %s was kept: %w", op.StashOID, err))
		}
		op.Phase = recovery.PhaseApplied
		f.Upsert(op)
		if err := recovery.Save(common, f); err != nil {
			return combine(bodyErr, err)
		}
		return combine(bodyErr, cleanup(common, id, op, &f))
	})
}

func cleanup(common string, id cmdid.ID, op recovery.Op, f *recovery.File) error {
	hits, err := listStashes()
	if err != nil {
		return err
	}
	ref := ""
	for _, hit := range hits {
		if hit.oid != op.StashOID {
			continue
		}
		if ref != "" {
			return fmt.Errorf("recovery stash %s is ambiguous; other stashes were not changed", op.StashOID)
		}
		ref = hit.ref
	}
	if ref != "" {
		if err := git.Verbose("stash", "drop", ref); err != nil {
			return err
		}
	}
	if err := deleteRecoveryRef(op); err != nil {
		return err
	}
	if !f.Remove(op.ID, op.StashOID) {
		return fmt.Errorf("recovery %s is not in the journal", op.ID)
	}
	if err := recovery.Save(common, *f); err != nil {
		return err
	}
	return clearStashField(id, op.Message)
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

func sourceSafe(op recovery.Op) bool {
	if op.Detached {
		oid, err := git.Output("rev-parse", "--verify", "HEAD")
		return err == nil && strings.TrimSpace(oid) == op.Source
	}
	if op.Source == "" {
		return false
	}
	return state.LocalBranchExists(op.Source)
}

func sourceKeptMessage(op recovery.Op) string {
	if op.Detached {
		return fmt.Sprintf("Uncommitted changes stay in the stash because HEAD is no longer %s. Inspect it with `eg repo doctor`.", op.Source)
	}
	return fmt.Sprintf("Uncommitted changes stay in the stash because the '%s' branch no longer exists. Inspect it with `eg repo doctor`.", op.Source)
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
		return fmt.Errorf("backup preparation cannot read submodule state: %w", err)
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
	if err := git.Verbose("stash", "pop", refs[0]); err != nil {
		return err
	}
	return cmdmem.Unset(id, cmdmem.FieldStash)
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
