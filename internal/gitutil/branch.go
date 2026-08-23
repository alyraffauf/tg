package gitutil

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// PullBase separates the immutable revision used to generate a patch from the
// canonical target branch stored in a pull request record.
type PullBase struct {
	Revision string
	Branch   string
}

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

// ResolveCommit resolves revision to its full commit SHA.
func (c *Client) ResolveCommit(ctx context.Context, dir, revision string) (string, error) {
	commit, err := c.gitOutput(ctx, dir, "rev-parse", "--verify", revision+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("resolve commit %q in %q: %w", revision, dir, err)
	}
	return strings.TrimSpace(string(commit)), nil
}

// ResolvePullBase resolves a local branch or configured remote-tracking
// branch to the commit used for patch generation and the target branch name.
func (c *Client) ResolvePullBase(ctx context.Context, dir, value string) (PullBase, error) {
	output, err := c.gitOutput(ctx, dir, "rev-parse", "--symbolic-full-name", "--verify", value)
	ref := strings.TrimSpace(string(output))
	if err != nil || ref == "" {
		return PullBase{}, fmt.Errorf("base %q is not an unambiguous local branch or configured remote-tracking branch", value)
	}

	branch, local := strings.CutPrefix(ref, "refs/heads/")
	if local && value != branch && value != ref {
		return PullBase{}, fmt.Errorf("base %q is not an unambiguous local branch or configured remote-tracking branch", value)
	}
	if !local {
		shortRef, remote := strings.CutPrefix(ref, "refs/remotes/")
		if !remote {
			return PullBase{}, fmt.Errorf("base %q is not an unambiguous local branch or configured remote-tracking branch", value)
		}
		branch, err = c.remoteBranch(ctx, dir, shortRef)
		if err != nil {
			return PullBase{}, err
		}
		if branch == "" || value != shortRef && value != ref {
			return PullBase{}, fmt.Errorf("base %q is not an unambiguous local branch or configured remote-tracking branch", value)
		}
	}
	if branch == "HEAD" {
		return PullBase{}, fmt.Errorf("base %q resolves to reserved branch name HEAD", value)
	}

	revision, err := c.ResolveCommit(ctx, dir, ref)
	if err != nil {
		return PullBase{}, err
	}
	return PullBase{Revision: revision, Branch: branch}, nil
}

// DefaultPullBase resolves origin's local default-branch reference.
func (c *Client) DefaultPullBase(ctx context.Context, dir string) (PullBase, error) {
	output, err := c.gitOutput(ctx, dir, "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD")
	if err != nil {
		return PullBase{}, fmt.Errorf("read origin default branch: %w", err)
	}
	ref := strings.TrimSpace(string(output))
	branch, found := strings.CutPrefix(ref, "refs/remotes/origin/")
	if !found || branch == "" {
		return PullBase{}, fmt.Errorf("origin default branch reference is invalid: %q", ref)
	}
	revision, err := c.ResolveCommit(ctx, dir, ref)
	if err != nil {
		return PullBase{}, err
	}
	return PullBase{Revision: revision, Branch: branch}, nil
}

// remoteBranch returns the branch portion of remoteAndBranch when its prefix
// is exactly one configured remote name.
func (c *Client) remoteBranch(ctx context.Context, dir, remoteAndBranch string) (string, error) {
	output, err := c.gitOutput(ctx, dir, "remote")
	if err != nil {
		return "", fmt.Errorf("list Git remotes: %w", err)
	}
	branch := ""
	for _, remote := range strings.Fields(string(output)) {
		candidate, found := strings.CutPrefix(remoteAndBranch, remote+"/")
		if !found || candidate == "" {
			continue
		}
		if branch != "" {
			return "", fmt.Errorf("base %q matches multiple configured remotes", remoteAndBranch)
		}
		branch = candidate
	}
	return branch, nil
}
