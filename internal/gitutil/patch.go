package gitutil

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// GeneratePatch returns a gzipped git format-patch series for commits in head
// that are not in base. The base commit must be an ancestor of head so the
// result can be applied onto the target branch with git am.
func GeneratePatch(ctx context.Context, repoDir, base, head string) ([]byte, error) {
	baseRevision, err := resolveBaseRevision(ctx, repoDir, base)
	if err != nil {
		return nil, fmt.Errorf("resolve base %q: %w", base, err)
	}
	headRevision, err := resolveRevision(ctx, repoDir, head)
	if err != nil {
		return nil, fmt.Errorf("resolve head %q: %w", head, err)
	}

	if err := gitCommand(ctx, repoDir, "merge-base", "--is-ancestor", baseRevision, headRevision); err != nil {
		return nil, fmt.Errorf("base %q is not an ancestor of head %q", base, head)
	}
	commitCount, err := gitOutput(ctx, repoDir, "rev-list", "--count", baseRevision+".."+headRevision)
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

	patch, err := gitOutput(ctx, repoDir, "format-patch", "--stdout", "--binary", "--full-index", baseRevision+".."+headRevision)
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

// DefaultBranch returns the branch named by origin's local HEAD reference.
func DefaultBranch(ctx context.Context, repoDir string) (string, error) {
	ref, err := gitOutput(ctx, repoDir, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
	if err != nil {
		return "", fmt.Errorf("read origin default branch: %w", err)
	}
	branch, found := strings.CutPrefix(strings.TrimSpace(string(ref)), "origin/")
	if !found || branch == "" {
		return "", fmt.Errorf("origin default branch reference is invalid: %q", strings.TrimSpace(string(ref)))
	}
	return branch, nil
}

func resolveRevision(ctx context.Context, repoDir, revision string) (string, error) {
	if _, err := gitOutput(ctx, repoDir, "rev-parse", "--verify", revision+"^{commit}"); err == nil {
		return revision, nil
	}

	remoteRevision := "origin/" + revision
	if _, err := gitOutput(ctx, repoDir, "rev-parse", "--verify", remoteRevision+"^{commit}"); err == nil {
		return remoteRevision, nil
	}
	return "", fmt.Errorf("commit does not exist locally or at origin")
}

func resolveBaseRevision(ctx context.Context, repoDir, revision string) (string, error) {
	remoteRevision := "origin/" + revision
	if _, err := gitOutput(ctx, repoDir, "rev-parse", "--verify", remoteRevision+"^{commit}"); err == nil {
		return remoteRevision, nil
	}
	return resolveRevision(ctx, repoDir, revision)
}

func gitCommand(ctx context.Context, repoDir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func gitOutput(ctx context.Context, repoDir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return output, nil
}
