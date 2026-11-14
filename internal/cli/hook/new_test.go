package hook

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/extsoft/elegant-git/internal/cli/catalog"
	"github.com/extsoft/elegant-git/internal/cli/sources"
	"github.com/extsoft/elegant-git/internal/prompt"
	"github.com/extsoft/elegant-git/internal/runtime"
)

func TestHookNewRejectsWorkTrack(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	sources.SetHookCommandIDsProvider(catalog.AllCanonicalCommandIDs)
	cmd := newNewCommand()
	ctx := prompt.WithPrompter(context.Background(), prompt.NewNonInteractive())
	ctx = runtime.WithRepoLayout(ctx, runtime.RepoLayout{RepoRoot: root})
	ctx = runtime.WithEditor(ctx, func(string) error { return nil })
	cmd.SetContext(ctx)
	err := newRun(cmd, []string{"work.track", "ahead", "common"})
	if err == nil || !strings.Contains(err.Error(), "work.track") {
		t.Fatalf("got %v", err)
	}
	hook := filepath.Join(root, ".config", "elegant-git", "hooks", "work-track-ahead")
	if _, statErr := os.Stat(hook); statErr == nil {
		t.Fatalf("created %s", hook)
	}
}
