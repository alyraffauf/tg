package gitutil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CurrentBranch returns the checked-out branch name at dir; errors if HEAD is
// detached.
func CurrentBranch(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("get current branch in %q: %w", dir, err)
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" || branch == "HEAD" {
		return "", fmt.Errorf("no current branch (detached HEAD) in %q", dir)
	}
	return branch, nil
}
