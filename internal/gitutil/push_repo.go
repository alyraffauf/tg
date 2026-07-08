package gitutil

import (
	"context"
	"fmt"
)

type PushNewRepoParams struct {
	Dir        string // local repository to push from
	Handle     string // Tangled owner handle
	Repo       string // repository name
	RemoteName string // git remote to add and push to
}

// PushNewRepo adds a remote at Dir and pushes the current branch.
// Fails if RemoteName already exists.
func PushNewRepo(ctx context.Context, params PushNewRepoParams) error {
	remoteURL := tangledRemoteURL(params.Handle, params.Repo)
	if err := runIn(params.Dir, ctx, "git", "remote", "add", params.RemoteName, remoteURL); err != nil {
		return fmt.Errorf("add remote %q (already exists? use --remote to pick another name): %w", params.RemoteName, err)
	}
	if err := runIn(params.Dir, ctx, "git", "push", "-u", params.RemoteName, "HEAD"); err != nil {
		return fmt.Errorf("push to %q: %w", params.RemoteName, err)
	}
	return nil
}
