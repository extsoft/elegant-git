package doctor

import (
	"fmt"
	"strings"

	"github.com/extsoft/elegant-git/internal/git"
	cmdmem "github.com/extsoft/elegant-git/internal/memory/cmd"
	memrepo "github.com/extsoft/elegant-git/internal/memory/repo"
	"github.com/extsoft/elegant-git/internal/recovery"
)

func diagnoseRecovery() ([]Finding, error) {
	gitDir, err := memrepo.GitDir()
	if err != nil {
		return nil, nil
	}
	common, err := memrepo.GitCommonDir()
	if err != nil {
		return nil, nil
	}
	file, err := recovery.Load(common)
	if err != nil {
		return nil, err
	}
	var out []Finding
	known := map[string]bool{}
	for _, op := range file.Ops {
		known[op.Ref] = true
		known[op.StashOID] = true
		op := op
		out = append(out, Finding{
			Problem: recoveryProblem(op),
			Repair:  "Clear this recovery record after you have restored the backup or decided to keep the stash. Elegant Git will not apply it or drop it.",
			Apply: func() error {
				return acknowledgeOp(common, op)
			},
		})
	}
	refs, err := recovery.RecoveryRefs()
	if err != nil {
		return nil, err
	}
	for _, ref := range refs {
		if known[ref.Name] || known[ref.OID] {
			continue
		}
		ref := ref
		out = append(out, Finding{
			Problem: fmt.Sprintf("Recovery ref %s points at %s and has no journal entry. Automatic cleanup is refused.", ref.Name, ref.OID),
			Repair:  fmt.Sprintf("Inspect it with `git show %s`. Do not delete it until you know what it saves.", ref.Name),
			Apply: func() error {
				return fmt.Errorf("refusing to delete %s automatically", ref.Name)
			},
		})
	}
	st, err := cmdmem.Load(gitDir)
	if err != nil {
		return nil, err
	}
	for id, entry := range st {
		if entry == nil || entry.Stash == "" {
			continue
		}
		if _, ok, err := file.Own(id, gitDir); err != nil {
			return nil, err
		} else if ok {
			continue
		}
		message := entry.Stash
		command := id
		found, err := legacyStashPresent(message)
		if err != nil {
			return nil, err
		}
		out = append(out, Finding{
			Problem: legacyProblem(command, message, found),
			Repair:  "Clear this recorded backup message. Elegant Git will not drop any stash.",
			Apply: func() error {
				return acknowledgeLegacy(common, gitDir, command, message)
			},
		})
	}
	return out, nil
}

func recoveryProblem(op recovery.Op) string {
	source := op.Source
	if op.Detached {
		source = "detached " + source
	}
	backup := op.StashOID
	manual := "The backup has not been identified yet."
	if backup != "" {
		manual = "To restore it yourself: git stash apply --index " + backup + "."
	} else {
		backup = "not identified yet"
	}
	ref := op.Ref
	if ref == "" {
		ref = "no private ref yet"
	}
	owned := ""
	if op.RebaseHead != "" {
		owned = " Owned rebase is " + op.RebaseHead + "."
	}
	return fmt.Sprintf("Recovery for %s is %s (source %s, backup %s, ref %s). %s%s A phase of applying is never applied again automatically.",
		op.Command, op.Phase, source, backup, ref, manual, owned)
}

func legacyStashPresent(message string) (bool, error) {
	hits, err := stashSubjects()
	if err != nil {
		return false, err
	}
	for _, hit := range hits {
		if recovery.StashMessage(hit.subject) == message {
			return true, nil
		}
	}
	return false, nil
}

func legacyProblem(command, message string, found bool) string {
	if !found {
		return fmt.Sprintf("Recorded backup for %s is %q. That stash was not found; other stashes were not changed.", command, message)
	}
	return fmt.Sprintf("Recorded backup for %s is %q. It is still in the stash list. Acknowledge it here before prune can offer it. Elegant Git will not drop it.", command, message)
}

func acknowledgeOp(common string, op recovery.Op) error {
	if err := deleteRecoveryRef(op); err != nil {
		return err
	}
	if err := recovery.Forget(common, op.ID, op.StashOID); err != nil {
		return err
	}
	if op.Worktree == "" || op.Message == "" {
		return nil
	}
	return clearRecordedStash(op.Worktree, op.Command, op.Message)
}

func acknowledgeLegacy(common, gitDir, command, message string) error {
	hits, err := stashSubjects()
	if err != nil {
		return err
	}
	for _, hit := range hits {
		if recovery.StashMessage(hit.subject) != message {
			continue
		}
		if err := recovery.WithLock(common, func() error {
			file, err := recovery.Load(common)
			if err != nil {
				return err
			}
			if hit.oid != "" && !contains(file.Acknowledged, hit.oid) {
				file.Acknowledged = append(file.Acknowledged, hit.oid)
			}
			return recovery.Save(common, file)
		}); err != nil {
			return err
		}
		break
	}
	return clearRecordedStash(gitDir, command, message)
}

func clearRecordedStash(gitDir, command, message string) error {
	st, err := cmdmem.Load(gitDir)
	if err != nil {
		return err
	}
	entry := st[command]
	if entry == nil || entry.Stash == "" {
		return nil
	}
	if message != "" && entry.Stash != message {
		return nil
	}
	entry.Stash = ""
	if entry.Branch == "" {
		delete(st, command)
	}
	return cmdmem.Save(gitDir, st)
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

type stashSubject struct {
	oid     string
	subject string
}

func stashSubjects() ([]stashSubject, error) {
	out, err := git.Output("stash", "list", "--format=%H%x00%s")
	if err != nil {
		return nil, err
	}
	var hits []stashSubject
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		oid, subject, ok := strings.Cut(line, "\x00")
		if !ok {
			return nil, fmt.Errorf("recovery cannot read the stash list")
		}
		hits = append(hits, stashSubject{oid: oid, subject: subject})
	}
	return hits, nil
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
