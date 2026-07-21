package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/alyraffauf/tg/internal/gitutil"
)

func parseHandleRepo(arg string) (string, string, error) {
	parts := strings.SplitN(arg, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.Contains(parts[1], "/") {
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
// keyed by.
func findRepoDid(ctx context.Context, handle, repo string) (string, error) {
	record, err := resolveRepoRecord(ctx, handle, repo)
	if err != nil {
		return "", err
	}
	return record.Value.RepoDid, nil
}
