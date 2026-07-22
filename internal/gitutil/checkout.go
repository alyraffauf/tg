package gitutil

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// CheckoutPatchParams configures a local branch reconstructed from a patch.
type CheckoutPatchParams struct {
	RepoDir      string
	Branch       string
	TargetBranch string
	Patch        []byte
	Force        bool
}

// CheckoutPatch creates a branch at the current target branch and applies a
// pull request patch series to it.
func (c *Client) CheckoutPatch(ctx context.Context, params CheckoutPatchParams) error {
	if err := c.requireCleanWorktree(ctx, params.RepoDir); err != nil {
		return err
	}
	if err := c.validateBranch(ctx, params.RepoDir, params.Branch); err != nil {
		return fmt.Errorf("invalid checkout branch %q: %w", params.Branch, err)
	}
	if err := c.validateBranch(ctx, params.RepoDir, params.TargetBranch); err != nil {
		return fmt.Errorf("invalid target branch %q: %w", params.TargetBranch, err)
	}

	targetRef := "refs/remotes/origin/" + params.TargetBranch
	refspec := "+refs/heads/" + params.TargetBranch + ":" + targetRef
	if err := c.gitCommand(ctx, params.RepoDir, "fetch", "origin", refspec); err != nil {
		return fmt.Errorf("fetch target branch %q: %w", params.TargetBranch, err)
	}

	switchFlag := "-c"
	if params.Force {
		switchFlag = "-C"
	}
	if err := c.gitCommand(ctx, params.RepoDir, "switch", switchFlag, params.Branch, targetRef); err != nil {
		return fmt.Errorf("check out branch %q: %w", params.Branch, err)
	}
	if err := c.applyPatch(ctx, params.RepoDir, params.Patch); err != nil {
		return fmt.Errorf("apply pull request patch; resolve conflicts with git am --continue or undo with git am --abort: %w", err)
	}
	return nil
}

func CheckoutPatch(ctx context.Context, params CheckoutPatchParams) error {
	return defaultClient.CheckoutPatch(ctx, params)
}

func (c *Client) validateBranch(ctx context.Context, repoDir, branch string) error {
	_, err := c.gitOutput(ctx, repoDir, "check-ref-format", "--branch", branch)
	return err
}

func (c *Client) requireCleanWorktree(ctx context.Context, repoDir string) error {
	status, err := c.gitOutput(ctx, repoDir, "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("inspect worktree: %w", err)
	}
	if len(bytes.TrimSpace(status)) != 0 {
		return fmt.Errorf("worktree has uncommitted changes")
	}
	return nil
}

func (c *Client) applyPatch(ctx context.Context, repoDir string, patch []byte) error {
	cmd := exec.CommandContext(ctx, "git", "am", "--3way")
	cmd.Dir = repoDir
	cmd.Stdin = bytes.NewReader(patch)
	stdout, stderr := c.writers()
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("git am --3way: %w", err)
	}
	return nil
}
