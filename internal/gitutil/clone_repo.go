package gitutil

import (
	"context"
	"fmt"
)

func CloneRepo(ctx context.Context, handle, repo, repoDir string) error {
	url := fmt.Sprintf("git@tangled.org:%s/%s", handle, repo)
	return run(ctx, "git", "clone", url, repoDir)
}
