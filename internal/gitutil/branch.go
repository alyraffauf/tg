package gitutil

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CurrentBranch returns the checked-out branch name at dir; errors if HEAD is
// detached.

func (c *Client) CurrentBranch(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	_, stderr := c.writers()
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("get current branch in %q: %w", dir, err)
	}
	branch := strings.TrimSpace(out.String())
	if branch == "" || branch == "HEAD" {
		return "", fmt.Errorf("no current branch (detached HEAD) in %q", dir)
	}
	return branch, nil
}

func CurrentBranch(ctx context.Context, dir string) (string, error) {
	return defaultClient.CurrentBranch(ctx, dir)
}
