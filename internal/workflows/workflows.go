// Package workflows runs optional ahead/after hook scripts for a command.
package workflows

// Skip when true disables ahead/after hook execution (--no-workflows).
var Skip bool

// RunAhead executes personal and common workflow files for <command>-ahead.
func RunAhead(command string) {
	if Skip {
		return
	}
	// Implemented in task 0003-port-plugins.
}

// RunAfter executes personal and common workflow files for <command>-after.
func RunAfter(command string) {
	if Skip {
		return
	}
	// Implemented in task 0003-port-plugins.
}
