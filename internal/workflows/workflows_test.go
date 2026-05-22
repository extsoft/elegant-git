package workflows

import "testing"

func TestSkipDisablesHooks(t *testing.T) {
	Skip = true
	RunAhead("start-work")
	RunAfter("start-work")
	Skip = false
}
