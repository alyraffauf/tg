package gitutil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// RepoContext holds the handle and repo name parsed from a git remote URL.
type RepoContext struct {
	Handle string
	Repo   string
}

// DetectRepoFromCWD reads the "origin" remote URL in the current directory
// It supports the ssh format git@tangled.org:handle/repo.
func DetectRepoFromCWD(ctx context.Context) (*RepoContext, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("get remote origin URL: %w", err)
	}

	url := strings.TrimSpace(string(output))

	prefix := "git@tangled.org:"
	if !strings.HasPrefix(url, prefix) {
		return nil, fmt.Errorf(
			"remote URL %q does not look like a Tangled repo (expected %s<handle>/<repo>)",
			url, prefix,
		)
	}

	path := strings.TrimPrefix(url, prefix)
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("could not parse handle/repo from %q", url)
	}

	repoName := strings.TrimSuffix(parts[1], ".git")
	return &RepoContext{
		Handle: parts[0],
		Repo:   repoName,
	}, nil
}
