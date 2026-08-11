package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Worktree struct {
	Root string
	repo string
}

func New(ctx context.Context, repoPath, ref string) (*Worktree, error) {
	// Git insists the target dir not exist yet: make a temp *parent*,
	// point git at a child of it.
	parent, err := os.MkdirTemp("", "reviewd-wt-*")
	if err != nil {
		return nil, err
	}
	root := filepath.Join(parent, "wt")

	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "--detach", root, ref)
	cmd.Dir = repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(parent)
		return nil, fmt.Errorf("sandbox: worktree add: %w: %s", err, out)
	}
	return &Worktree{Root: root, repo: repoPath}, nil
}

func (w *Worktree) Close() error {
	cmd := exec.Command("git", "worktree", "remove", "--force", w.Root)
	cmd.Dir = w.repo
	gitErr := cmd.Run()
	// Belt and braces: remove the temp parent regardless, so a failed
	// git command never leaks directories across runs.
	rmErr := os.RemoveAll(filepath.Dir(w.Root))
	if gitErr != nil {
		return fmt.Errorf("sandbox: worktree remove: %w", gitErr)
	}
	return rmErr
}
