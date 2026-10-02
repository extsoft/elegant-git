package work

import (
	"context"
	"strings"

	"github.com/extsoft/elegant-git/internal/cli/argspec"
	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	"github.com/extsoft/elegant-git/internal/config"
	"github.com/extsoft/elegant-git/internal/state"
)

func syncBranchComplete(ctx context.Context) ([]argspec.Choice, error) {
	if state.AreThereRemotes() {
		cliruntime.FetchOrInform()
	}
	branch := cliruntime.CurrentBranch()
	seen := map[string]bool{}
	var out []argspec.Choice
	add := func(val, desc string) {
		val = strings.TrimSpace(val)
		if val == "" || seen[val] || !state.RefExists(val) {
			return
		}
		seen[val] = true
		out = append(out, argspec.Choice{Value: val, Description: desc})
	}
	add(config.FreshestBranchSourceBranch(branch), "Source branch.")
	add(config.FreshestDefaultBranch(), "Default development branch.")
	add(state.UpstreamOf(branch), "This branch's upstream.")
	rest, err := sources.BranchNamesListed(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range rest {
		if seen[c.Value] || syncBranchIsCurrent(branch, c.Value) {
			continue
		}
		seen[c.Value] = true
		out = append(out, c)
	}
	return out, nil
}

func syncBranchIsCurrent(currentBranch, ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" || currentBranch == "" {
		return false
	}
	return ref == currentBranch || ref == "origin/"+currentBranch
}
