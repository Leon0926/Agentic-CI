package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Leon0926/Agentic-CI/internal/agent"
)

const maxReadBytes = 64 * 1024

type ReadFile struct {
	root string // absolute; the worktree root
}

func NewReadFile(root string) (*ReadFile, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &ReadFile{root: abs}, nil
}

func (t *ReadFile) Def() agent.ToolDef {
	return agent.ToolDef{
		Name:        "read_file",
		Description: "Read a file from the repository. Paths are relative to the repo root.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
			"required": []string{"path"},
		},
	}
}

type readFileInput struct {
	Path string `json:"path"`
}

func (t *ReadFile) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in readFileInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	resolved, err := t.resolve(in.Path)
	if err != nil {
		return "", err // model error → loop turns it into IsError result
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", in.Path, err)
	}
	truncated := false
	if len(data) > maxReadBytes {
		data, truncated = data[:maxReadBytes], true
	}
	out := numberLines(string(data))
	if truncated {
		out += "\n[truncated]"
	}
	return out, nil
}

// resolve is the security boundary: reject absolute paths, join under
// root, and verify the result is still under root after cleaning.
func (t *ReadFile) resolve(p string) (string, error) {
	if filepath.IsAbs(p) {
		return "", fmt.Errorf("absolute paths not allowed: %s", p)
	}
	joined := filepath.Join(t.root, p) // Join also Cleans, collapsing ../
	if joined != t.root && !strings.HasPrefix(joined, t.root+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes repository root: %s", p)
	}
	return joined, nil
}

func numberLines(s string) string {
	lines := strings.Split(s, "\n")
	var b strings.Builder
	for i, line := range lines {
		fmt.Fprintf(&b, "%d\t%s\n", i+1, line)
	}
	return strings.TrimSuffix(b.String(), "\n")
}
