package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFile_Run(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package main\n"), 0o644))

	rf, err := NewReadFile(dir)
	require.NoError(t, err)

	t.Run("reads a real file with line numbers", func(t *testing.T) {
		out, err := rf.Run(context.Background(), json.RawMessage(`{"path":"hello.go"}`))
		require.NoError(t, err)
		assert.Contains(t, out, "1\tpackage main")
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		_, err := rf.Run(context.Background(), json.RawMessage(`{"path":"../../etc/passwd"}`))
		assert.Error(t, err)
	})

	t.Run("rejects absolute paths", func(t *testing.T) {
		_, err := rf.Run(context.Background(), json.RawMessage(`{"path":"/etc/passwd"}`))
		assert.Error(t, err)
	})

	t.Run("missing file is a model-recoverable error", func(t *testing.T) {
		_, err := rf.Run(context.Background(), json.RawMessage(`{"path":"nope.go"}`))
		assert.Error(t, err)
	})
}
