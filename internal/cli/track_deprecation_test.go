package cli

import (
	"bytes"
	"os"
	"testing"

	workcmd "github.com/extsoft/elegant-git/internal/cli/work"
	"github.com/extsoft/elegant-git/internal/deprecation"
)

func TestWorkTrackDEP021Once(t *testing.T) {
	c := workcmd.NewCommand()
	for _, sub := range c.Commands() {
		if sub.Name() != "track" {
			continue
		}
		deprecation.Reset()
		var buf bytes.Buffer
		deprecation.SetOutputWriter(&buf)
		t.Cleanup(func() {
			deprecation.SetOutputWriter(os.Stderr)
			deprecation.Reset()
		})
		recordDeprecatedSurface(sub)
		recordDeprecatedSurface(sub)
		deprecation.Flush()
		want := "warning: work track is deprecated; use work switch.\n"
		if buf.String() != want {
			t.Fatalf("stderr %q", buf.String())
		}
		return
	}
	t.Fatal("work track command missing")
}
