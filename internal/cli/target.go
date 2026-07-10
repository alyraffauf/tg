package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/alyraffauf/tg/internal/gitutil"
)

func parseHandleRepo(arg string) (string, string, error) {
	parts := strings.SplitN(arg, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected handle/repo, got %q", arg)
	}
	return parts[0], parts[1], nil
}

// resolveTarget returns the handle and repo name from an explicit
// "handle/repo" argument or by detecting the git remote in the CWD.
func resolveTarget(ctx context.Context, args []string) (string, string, error) {
	if len(args) == 1 {
		return parseHandleRepo(args[0])
	}

	rc, err := gitutil.DetectRepoFromCWD(ctx)
	if err != nil {
		return "", "", fmt.Errorf("detect repo from current directory: %w", err)
	}
	return rc.Handle, rc.Repo, nil
}

// findRepoDid resolves handle/repo to the repo's repoDid, which listIssues is
// keyed by. It looks the record up directly by name (current schema uses the
// name as the rkey), falling back to a listing for legacy repos whose rkey is a
// TID with the name in the body.
func findRepoDid(ctx context.Context, handle, repo string) (string, error) {
	ident, err := resolver.ResolveHandle(ctx, handle)
	if err != nil {
		return "", fmt.Errorf("resolve handle %q: %w", handle, err)
	}

	repoURI := fmt.Sprintf("at://%s/sh.tangled.repo/%s", ident.DID, repo)
	if got, err := client.GetRepo(ctx, repoURI); err == nil {
		return got.Value.RepoDid, nil
	}

	if repos, err := client.ListRepos(ctx, ident.DID.String()); err == nil {
		for _, candidate := range repos.Items {
			if candidate.Value.Name == repo || strings.HasSuffix(candidate.URI, "/"+repo) {
				return candidate.Value.RepoDid, nil
			}
		}
	}

	return "", fmt.Errorf("repo %q not found for handle %q", repo, handle)
}
