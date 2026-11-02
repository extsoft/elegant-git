package work

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/extsoft/elegant-git/internal/git"
	"github.com/extsoft/elegant-git/internal/state"
	"github.com/extsoft/elegant-git/internal/text"
)

func rebaseOnto(target string) error {
	return rebaseTarget(target, true)
}

func rebaseHoldingAside(target string) error {
	return rebaseTarget(target, false)
}

func rebaseTarget(target string, restore bool) error {
	base := state.AlreadyAppliedBase(target)
	return asideAround(upstreamFor(base, target), target, restore, func() error {
		if base == "" {
			return git.Verbose("rebase", target)
		}
		if n := appliedPrefixLen(target, base); n > 0 {
			text.InfoText(skipMessage(n, target))
		}
		return git.Verbose("rebase", "--onto", target, base)
	})
}

func asideAround(upstream, target string, restore bool, run func() error) error {
	if !state.IsThereActiveRebase() {
		if err := restoreUntrackedAside(); err != nil {
			return err
		}
	}
	paths, err := collidingPaths(upstream, target)
	if err != nil {
		return err
	}
	if err := moveAside(paths); err != nil {
		return err
	}
	if err := run(); err != nil {
		if len(paths) > 0 {
			text.InfoText(asideHint)
		}
		return err
	}
	if restore {
		return restoreAside()
	}
	return nil
}

func continueRebase() error {
	return git.Verbose("rebase", "--continue")
}

func appliedPrefixLen(target, base string) int {
	mb, err := git.Output("merge-base", target, "HEAD")
	mb = strings.TrimSpace(mb)
	if i := strings.IndexByte(mb, '\n'); i >= 0 {
		mb = strings.TrimSpace(mb[:i])
	}
	if err != nil || mb == "" {
		return 0
	}
	out, err := git.Output("rev-list", "--count", mb+".."+base)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0
	}
	return n
}

func skipMessage(n int, target string) string {
	noun := "commits"
	if n == 1 {
		noun = "commit"
	}
	return fmt.Sprintf("Skipping %d %s already in %s.", n, noun, target)
}
