package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initGitFixture creates a tiny real git repo so grep_repo (which shells out
// to `git grep`) has something to search — no mocking of git itself.
func initGitFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	run("init", "-q")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "test")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.go"),
		[]byte("var Key = \"AKIAIOSFODNN7EXAMPLE\"\n"), 0o644))
	run("add", "-A")
	run("commit", "-q", "-m", "seed")
	return dir
}

func TestGrepRepo_Run(t *testing.T) {
	dir := initGitFixture(t)
	gr, err := NewGrepRepo(dir)
	require.NoError(t, err)

	t.Run("finds a real match", func(t *testing.T) {
		out, err := gr.Run(context.Background(), json.RawMessage(`{"pattern":"AKIA[0-9A-Z]{16}"}`))
		require.NoError(t, err)
		assert.Contains(t, out, "config.go")
		assert.Contains(t, out, "AKIAIOSFODNN7EXAMPLE")
	})

	t.Run("no matches is not an error", func(t *testing.T) {
		out, err := gr.Run(context.Background(), json.RawMessage(`{"pattern":"NOPE_NOT_THERE"}`))
		require.NoError(t, err)
		assert.Equal(t, "no matches found", out)
	})

	t.Run("empty pattern rejected", func(t *testing.T) {
		_, err := gr.Run(context.Background(), json.RawMessage(`{"pattern":""}`))
		assert.Error(t, err)
	})

	t.Run("invalid regex rejected", func(t *testing.T) {
		_, err := gr.Run(context.Background(), json.RawMessage(`{"pattern":"("}`))
		assert.Error(t, err)
	})
}
