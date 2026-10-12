//go:build windows

package workspace

import (
	"os"
	"os/exec"
	"os/signal"
)

func watchChildSignals(cmd *exec.Cmd) func() {
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt)

	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-sigCh:
			case <-done:
				return
			}
		}
	}()

	return func() {
		close(done)
		signal.Stop(sigCh)
	}
}

func signalExitCode(exitErr *exec.ExitError) (int, bool) {
	if code := exitErr.ExitCode(); code >= 0 {
		return code, true
	}
	return 0, false
}
