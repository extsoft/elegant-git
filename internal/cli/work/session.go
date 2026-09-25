package work

import (
	"fmt"

	cliruntime "github.com/extsoft/elegant-git/internal/cli/runtime"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var dispatchAction = dispatch

// Detected is the work detection result for a following picker.
// Ask is true when the caller should still prompt. Ran is the action already run
// when detection still asks. Accept is the branch argument for a later work accept,
// from the same snapshot as Actions.
type Detected struct {
	Actions []string
	Ask     bool
	Ran     string
	Accept  []string
}

// RunDetected runs the action detection chooses for this repository.
func RunDetected(cmd *cobra.Command) (Detected, error) {
	snap := inspect()
	ask, ran, err := dispatchDetected(cmd, detect(snap))
	if err != nil || !ask {
		return Detected{Ask: ask, Ran: ran}, err
	}
	if ran != "" {
		snap = inspect()
	}
	return Detected{
		Actions: relevantFrom(snap),
		Ask:     true,
		Ran:     ran,
		Accept:  acceptArgs(snap),
	}, nil
}

func runSession(cmd *cobra.Command, inspectFn func() snapshot) error {
	snap := inspectFn()
	d := detect(snap)
	if d.Action == "" {
		printEval(d.Steps)
	}
	ask, _, err := dispatchDetected(cmd, d)
	if err != nil || !ask {
		return err
	}
	if d.Action != "" {
		snap = inspectFn()
	}
	return askOnce(cmd, snap)
}

func dispatchDetected(cmd *cobra.Command, d outcome) (ask bool, ran string, err error) {
	if d.Action == "" {
		return true, "", nil
	}
	printEval(d.Steps)
	if err := dispatchAction(cmd, d.Action); err != nil {
		return false, d.Action, err
	}
	if !d.ThenAsk {
		return false, d.Action, nil
	}
	text.PlainText("selected: ask")
	return true, d.Action, nil
}

func askOnce(cmd *cobra.Command, snap snapshot) error {
	p := prompt.FromContext(cmd.Context())
	ans, err := cliruntime.PickActionOrHelp(cmd, p, "What now", askChoices(snap), "quit")
	if err != nil {
		return err
	}
	if ans == "" || ans == "quit" {
		return nil
	}
	var args []string
	if ans == "accept" {
		args = acceptArgs(snap)
	}
	return dispatchAction(cmd, ans, args...)
}

func printEval(steps []string) {
	text.CommandText(detectionTitle)
	for _, s := range steps {
		text.PlainText(s)
	}
}

func dispatch(cmd *cobra.Command, action string, args ...string) error {
	if action == "quit" {
		return nil
	}
	sub, _, err := cmd.Find([]string{action})
	if err != nil {
		return cliruntime.NewUsageError(cmd, err)
	}
	if sub == cmd {
		return cliruntime.NewUsageError(cmd, fmt.Errorf("unknown work action %q", action))
	}
	return cliruntime.RunBound(cmd, sub, args)
}
