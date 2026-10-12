//go:build !windows

package workspace

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func watchChildSignals(cmd *exec.Cmd) func() {
	sigCh := make(chan os.Signal, 4)
	forward := []os.Signal{syscall.SIGTERM}
	swallow := []os.Signal{syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP}
	signal.Notify(sigCh, append(forward, swallow...)...)

	done := make(chan struct{})
	go func() {
		for {
			select {
			case sig := <-sigCh:
				if sig == syscall.SIGTERM && cmd.Process != nil {
					_ = cmd.Process.Signal(sig)
				}
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
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return 0, false
	}
	return 128 + int(status.Signal()), true
}
