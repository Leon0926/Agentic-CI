package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Leon0926/Agentic-CI/internal/agent"
)

const maxGrepLines = 200

// GrepRepo is the "grep_repo" tool: lets the model search the repo with a
// regex pattern via `git grep`, scoped to the worktree root.
type GrepRepo struct {
	root string
}

// NewGrepRepo builds a GrepRepo tool rooted at the given directory.
// Converts root to an absolute path since it's used as the cwd for the
// `git grep` subprocess.
func NewGrepRepo(root string) (*GrepRepo, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &GrepRepo{root: abs}, nil
}

// Def describes this tool to the model: its name, what it does, and the
// JSON schema for its input (a single "pattern" string).
func (t *GrepRepo) Def() agent.ToolDef {
	return agent.ToolDef{
		Name:        "grep_repo",
		Description: "Search the repository for a regex pattern. Returns matching lines as path:line:content.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string"},
			},
			"required": []string{"pattern"},
		},
	}
}

// grepRepoInput matches the JSON schema declared in Def().
type grepRepoInput struct {
	Pattern string `json:"pattern"`
}

// Run is the actual tool call: validate the pattern, shell out to
// `git grep`, and return matching lines (or a friendly "no matches").
func (t *GrepRepo) Run(ctx context.Context, input json.RawMessage) (string, error) {
	// 1. Parse the model's JSON args.
	var in grepRepoInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if in.Pattern == "" {
		return "", errors.New("pattern must not be empty")
	}
	// 2. Fail fast on bad regex instead of letting git grep error out cryptically.
	if _, err := regexp.Compile(in.Pattern); err != nil {
		return "", fmt.Errorf("invalid regex: %w", err)
	}

	// 3. Run `git grep -n -E -e <pattern>` in the repo root.
	//    -n = show line numbers, -E = extended regex, -e = pattern arg.
	cmd := exec.CommandContext(ctx, "git", "grep", "-n", "-E", "-e", in.Pattern)
	cmd.Dir = t.root
	out, err := cmd.Output()

	if err != nil {
		// git grep exits 1 when there are simply no matches (not a real error).
		// Any other exit code/err is a genuine failure.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "no matches found", nil // exit 1 = clean "no hits"
		}
		return "", fmt.Errorf("git grep failed: %w", err)
	}

	// 4. Cap output so one huge match set can't flood the model's context.
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) > maxGrepLines {
		lines = lines[:maxGrepLines]
		lines = append(lines, "[truncated]")
	}
	return strings.Join(lines, "\n"), nil
}
