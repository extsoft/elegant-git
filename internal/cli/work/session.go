package work

import (
	"fmt"

	cliruntime "github.com/bees-hive/elegant-git/internal/cli/runtime"
	"github.com/bees-hive/elegant-git/internal/prompt"
	"github.com/bees-hive/elegant-git/internal/text"
	"github.com/spf13/cobra"
)

var dispatchAction = dispatch

func runSession(cmd *cobra.Command, inspectFn func() snapshot) error {
	snap := inspectFn()
	d := detect(snap)
	printEval(d.Steps)
	if d.Action != "" && !d.ThenAsk {
		return dispatchAction(cmd, d.Action)
	}
	if d.Action != "" {
		if err := dispatchAction(cmd, d.Action); err != nil {
			return err
		}
		text.PlainText("selected: ask")
		snap = inspectFn()
	}
	return askOnce(cmd, snap)
}

func askOnce(cmd *cobra.Command, snap snapshot) error {
	p := prompt.FromContext(cmd.Context())
	ans, err := p.Closed("What next?", askOptions(snap), "quit", true)
	if err != nil {
		return err
	}
	if ans == "" || ans == "quit" {
		return nil
	}
	return dispatchAction(cmd, ans)
}

func printEval(steps []string) {
	text.CommandText(detectionTitle)
	for _, s := range steps {
		text.PlainText(s)
	}
}

func dispatch(cmd *cobra.Command, action string) error {
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
	sub.SetContext(cmd.Context())
	sub.SetOut(cmd.OutOrStdout())
	sub.SetErr(cmd.ErrOrStderr())
	if sub.RunE != nil {
		return sub.RunE(sub, nil)
	}
	if sub.Run != nil {
		sub.Run(sub, nil)
	}
	return nil
}
