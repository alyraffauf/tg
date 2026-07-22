package gitutil

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// GeneratePatch returns a gzipped git format-patch series for commits in head
// that are not in base. The base commit must be an ancestor of head so the
// result can be applied onto the target branch with git am.
func (c *Client) GeneratePatch(ctx context.Context, repoDir, base, head string) ([]byte, error) {
	baseRevision, err := c.resolveBaseRevision(ctx, repoDir, base)
	if err != nil {
		return nil, fmt.Errorf("resolve base %q: %w", base, err)
	}
	headRevision, err := c.resolveRevision(ctx, repoDir, head)
	if err != nil {
		return nil, fmt.Errorf("resolve head %q: %w", head, err)
	}

	if err := c.gitCommand(ctx, repoDir, "merge-base", "--is-ancestor", baseRevision, headRevision); err != nil {
		return nil, fmt.Errorf("base %q is not an ancestor of head %q", base, head)
	}
	commitCount, err := c.gitOutput(ctx, repoDir, "rev-list", "--count", baseRevision+".."+headRevision)
	if err != nil {
		return nil, fmt.Errorf("count commits from %q to %q: %w", base, head, err)
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(commitCount)))
	if err != nil {
		return nil, fmt.Errorf("parse commit count from %q to %q: %w", base, head, err)
	}
	if count == 0 {
		return nil, fmt.Errorf("no commits between base %q and head %q", base, head)
	}

	patch, err := c.gitOutput(ctx, repoDir, "format-patch", "--stdout", "--binary", "--full-index", baseRevision+".."+headRevision)
	if err != nil {
		return nil, fmt.Errorf("create patch from %q to %q: %w", base, head, err)
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(patch); err != nil {
		return nil, fmt.Errorf("compress patch: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finish patch compression: %w", err)
	}
	return compressed.Bytes(), nil
}

func GeneratePatch(ctx context.Context, repoDir, base, head string) ([]byte, error) {
	return defaultClient.GeneratePatch(ctx, repoDir, base, head)
}

// DefaultBranch returns the branch named by origin's local HEAD reference.
func (c *Client) DefaultBranch(ctx context.Context, repoDir string) (string, error) {
	ref, err := c.gitOutput(ctx, repoDir, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
	if err != nil {
		return "", fmt.Errorf("read origin default branch: %w", err)
	}
	branch, found := strings.CutPrefix(strings.TrimSpace(string(ref)), "origin/")
	if !found || branch == "" {
		return "", fmt.Errorf("origin default branch reference is invalid: %q", strings.TrimSpace(string(ref)))
	}
	return branch, nil
}

func DefaultBranch(ctx context.Context, repoDir string) (string, error) {
	return defaultClient.DefaultBranch(ctx, repoDir)
}

func (c *Client) resolveRevision(ctx context.Context, repoDir, revision string) (string, error) {
	if _, err := c.gitOutput(ctx, repoDir, "rev-parse", "--verify", revision+"^{commit}"); err == nil {
		return revision, nil
	}

	remoteRevision := "origin/" + revision
	if _, err := c.gitOutput(ctx, repoDir, "rev-parse", "--verify", remoteRevision+"^{commit}"); err == nil {
		return remoteRevision, nil
	}
	return "", fmt.Errorf("commit does not exist locally or at origin")
}

func (c *Client) resolveBaseRevision(ctx context.Context, repoDir, revision string) (string, error) {
	remoteRevision := "origin/" + revision
	if _, err := c.gitOutput(ctx, repoDir, "rev-parse", "--verify", remoteRevision+"^{commit}"); err == nil {
		return remoteRevision, nil
	}
	return c.resolveRevision(ctx, repoDir, revision)
}

func (c *Client) gitCommand(ctx context.Context, repoDir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoDir
	var output bytes.Buffer
	stdout, stderr := c.writers()
	cmd.Stdout = io.MultiWriter(stdout, &output)
	cmd.Stderr = io.MultiWriter(stderr, &output)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(output.String()))
	}
	return nil
}

func (c *Client) gitOutput(ctx context.Context, repoDir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoDir
	var output bytes.Buffer
	cmd.Stdout = &output
	_, stderr := c.writers()
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return output.Bytes(), nil
}
