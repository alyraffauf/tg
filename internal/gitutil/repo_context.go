package gitutil

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"slices"
	"strings"

	"github.com/alyraffauf/tg/knot"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// HostedGitHost is Tangled's hosted Git proxy.
const HostedGitHost = "tangled.org"

// defaultRemote is the conventional name of the primary git remote.
const defaultRemote = "origin"

// knotRemoteURL builds an SSH URL for a repository on knotHost. Tangled's
// default Knot and callers without a selected Knot use the hosted proxy.
// repoPath may be a handle/repo pair or a repository DID.
func knotRemoteURL(knotHost string, sshPort int, repoPath string) string {
	gitHost := knotHost
	if gitHost == "" || gitHost == knot.DefaultKnot {
		gitHost = HostedGitHost
	}
	if sshPort != 22 {
		return fmt.Sprintf("ssh://git@%s:%d/%s", gitHost, sshPort, repoPath)
	}
	return "git@" + gitHost + ":" + repoPath
}

type repositoryDIDRemote struct {
	Protocol string
	KnotHost string
	SSHPort  int
	RepoDID  string
}

func repositoryDIDRemoteURL(remote repositoryDIDRemote) (string, error) {
	if _, err := syntax.ParseDID(remote.RepoDID); err != nil {
		return "", fmt.Errorf("invalid repository DID %q: %w", remote.RepoDID, err)
	}
	switch remote.Protocol {
	case "ssh":
		return knotRemoteURL(remote.KnotHost, remote.SSHPort, remote.RepoDID), nil
	case "https":
		host := remote.KnotHost
		if host == "" || host == knot.DefaultKnot {
			host = HostedGitHost
		}
		return "https://" + host + "/" + remote.RepoDID, nil
	default:
		return "", fmt.Errorf("unsupported clone protocol %q", remote.Protocol)
	}
}

// RepoContext holds an untrusted repository candidate parsed from a git remote
// URL. KnotHost is empty for Tangled's hosted Git endpoint; otherwise callers
// must verify it against the repository record before trusting the candidate.
type RepoContext struct {
	KnotHost string
	Handle   string
	Repo     string
	RepoDID  string
}

// DetectRepoCandidatesFromCWD scans the git remotes in the current directory
// for hosted Tangled URLs and untrusted custom-Knot SSH or HTTPS candidates,
// checking the default remote first.
func (c *Client) DetectRepoCandidatesFromCWD(ctx context.Context) ([]RepoContext, error) {
	remotes, err := c.gitLines(ctx, "remote")
	if err != nil {
		return nil, fmt.Errorf("list git remotes: %w", err)
	}

	var candidates []RepoContext
	for _, name := range originFirst(remotes) {
		urls, err := c.gitLines(ctx, "remote", "get-url", "--all", name)
		if err != nil {
			return nil, fmt.Errorf("get URLs for remote %q: %w", name, err)
		}
		for _, raw := range urls {
			if candidate, ok := parseRepoCandidate(raw); ok {
				candidates = append(candidates, *candidate)
			}
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no Tangled remote candidate found among %d remote(s) %q; pass the repository as handle/repo", len(remotes), remotes)
	}
	return candidates, nil
}

func DetectRepoCandidatesFromCWD(ctx context.Context) ([]RepoContext, error) {
	return defaultClient.DetectRepoCandidatesFromCWD(ctx)
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

// parseRepoCandidate parses a hosted Tangled URL or custom SSH/HTTPS URL into
// an untrusted repository candidate. Custom hosts must be verified by the app
// layer against the repository record.
//
// Tangled's hosted endpoint supports SCP-like, ssh://, git://, https://, and
// http:// URLs. Custom Knot candidates may use SSH or HTTPS.
func parseRepoCandidate(raw string) (*RepoContext, bool) {
	u, err := parseGitURL(strings.TrimSpace(raw))
	if err != nil {
		return nil, false
	}
	hosted := strings.EqualFold(u.Hostname(), HostedGitHost)
	if !hosted && u.Scheme != "ssh" && u.Scheme != "https" {
		return nil, false
	}
	path := strings.Trim(strings.TrimPrefix(u.Path, "/"), "/")
	candidate, ok := splitRepoPath(path)
	if !ok {
		return nil, false
	}
	if !hosted {
		candidate.KnotHost = strings.ToLower(u.Hostname())
	}
	return candidate, true
}

func splitRepoPath(path string) (*RepoContext, bool) {
	if strings.Contains(path, "/") {
		return splitHandleRepo(path)
	}
	if _, err := syntax.ParseDID(path); err != nil {
		return nil, false
	}
	return &RepoContext{RepoDID: path}, true
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
	for line := range strings.SplitSeq(captured.String(), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}
