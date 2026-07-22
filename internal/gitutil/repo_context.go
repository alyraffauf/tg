package gitutil

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"slices"
	"strings"
)

// tangledHost is the hostname for Tangled repositories.
const tangledHost = "tangled.org"

// defaultRemote is the conventional name of the primary git remote.
const defaultRemote = "origin"

// tangledRemoteURL builds the SSH clone/push URL for a Tangled repo.
func tangledRemoteURL(handle, repo string) string {
	return "git@" + tangledHost + ":" + handle + "/" + repo
}

// RepoContext holds the handle and repo name parsed from a git remote URL.
type RepoContext struct {
	Handle string
	Repo   string
}

// DetectRepoFromCWD scans the git remotes in the current directory for one
// pointing at Tangled, checking the default remote first. Returns the first
// match.
func (c *Client) DetectRepoFromCWD(ctx context.Context) (*RepoContext, error) {
	remotes, err := c.gitLines(ctx, "remote")
	if err != nil {
		return nil, fmt.Errorf("list git remotes: %w", err)
	}

	for _, name := range originFirst(remotes) {
		urls, err := c.gitLines(ctx, "remote", "get-url", "--all", name)
		if err != nil {
			return nil, fmt.Errorf("get URLs for remote %q: %w", name, err)
		}
		for _, raw := range urls {
			if rc, ok := parseTangledURL(raw); ok {
				return rc, nil
			}
		}
	}

	return nil, fmt.Errorf("no Tangled remote found among %d remote(s) %q; pass the repository as handle/repo", len(remotes), remotes)
}

func DetectRepoFromCWD(ctx context.Context) (*RepoContext, error) {
	return defaultClient.DetectRepoFromCWD(ctx)
}

// originFirst returns remotes with the default remote first (if present),
// followed by the rest in their original order.
func originFirst(remotes []string) []string {
	idx := slices.Index(remotes, defaultRemote)
	if idx <= 0 {
		return remotes
	}
	ordered := make([]string, 0, len(remotes))
	ordered = append(ordered, defaultRemote)
	ordered = append(ordered, remotes[:idx]...)
	ordered = append(ordered, remotes[idx+1:]...)
	return ordered
}

// parseTangledURL parses a Tangled git remote URL into handle and repo.
// Returns ok=false for URLs that don't point at Tangled.
//
// Supported formats: SCP-like (git@tangled.org:handle/repo), ssh://, git://,
// https://, and http:// URLs.
func parseTangledURL(raw string) (*RepoContext, bool) {
	u, err := parseGitURL(strings.TrimSpace(raw))
	if err != nil {
		return nil, false
	}
	if !strings.EqualFold(u.Hostname(), tangledHost) {
		return nil, false
	}
	return splitHandleRepo(strings.TrimPrefix(u.Path, "/"))
}

// parseGitURL parses a git remote URL, including SCP-like syntax
// (e.g. git@host:path), which net/url.Parse does not handle.
func parseGitURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, errors.New("empty URL")
	}
	// Schemed URLs (ssh://, git://, https://, etc.) are handled by stdlib.
	if strings.Contains(raw, "://") {
		return url.Parse(raw)
	}
	// SCP-like: [user@]host:path, where ':' comes before any '/'.
	colon := strings.Index(raw, ":")
	slash := strings.Index(raw, "/")
	if colon > 0 && (slash < 0 || colon < slash) {
		user, host, hasUser := strings.Cut(raw[:colon], "@")
		if !hasUser {
			host, user = user, ""
		}
		var builder strings.Builder
		builder.WriteString("ssh://")
		if user != "" {
			builder.WriteString(user)
			builder.WriteString("@")
		}
		builder.WriteString(host)
		builder.WriteString("/")
		builder.WriteString(raw[colon+1:])
		return url.Parse(builder.String())
	}
	// Local path or anything else — let stdlib produce the error.
	return url.Parse(raw)
}

// splitHandleRepo splits "handle/repo" (optionally with a trailing .git).
// Leading/trailing slashes are tolerated; returns ok=false for empty
// segments or paths with extra segments.
func splitHandleRepo(path string) (*RepoContext, bool) {
	path = strings.Trim(path, "/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, false
	}
	if strings.Contains(parts[1], "/") {
		return nil, false
	}
	return &RepoContext{
		Handle: parts[0],
		Repo:   strings.TrimSuffix(parts[1], ".git"),
	}, true
}

// gitLines runs git with the given args and returns non-empty output lines.
func (c *Client) gitLines(ctx context.Context, args ...string) ([]string, error) {
	// Output is intentionally captured; diagnostics still go to the client's sink.
	cmd := exec.CommandContext(ctx, "git", args...)
	var captured strings.Builder
	cmd.Stdout = &captured
	_, stderr := c.writers()
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	var lines []string
	for _, line := range strings.Split(captured.String(), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func gitLines(ctx context.Context, args ...string) ([]string, error) {
	return defaultClient.gitLines(ctx, args...)
}
