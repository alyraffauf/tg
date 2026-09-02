package gitutil

import (
	"context"
	"fmt"
	"strings"
)

type PushNewRepoParams struct {
	Dir        string // local repository to push from
	KnotHost   string // explicit Knot hosting the repository
	SSHPort    int    // Knot SSH port
	RepoDID    string // stable repository DID, when available
	RemoteName string // git remote to add and push to
}

// PushNewRepo adds a remote at Dir and pushes the current branch.
// Fails if RemoteName already exists.
func (c *Client) PushNewRepo(ctx context.Context, params PushNewRepoParams) error {
	if params.RemoteName == "" || strings.HasPrefix(params.RemoteName, "-") || strings.ContainsAny(params.RemoteName, "\x00\r\n") {
		return fmt.Errorf("invalid remote name %q", params.RemoteName)
	}
	remoteURL, err := repositoryDIDRemoteURL(repositoryDIDRemote{
		Protocol: "ssh",
		KnotHost: params.KnotHost,
		SSHPort:  params.SSHPort,
		RepoDID:  params.RepoDID,
	})
	if err != nil {
		return err
	}
	if err := c.runIn(params.Dir, ctx, "git", "remote", "add", params.RemoteName, remoteURL); err != nil {
		return fmt.Errorf("add remote %q (already exists? use --remote to pick another name): %w", params.RemoteName, err)
	}
	if err := c.runIn(params.Dir, ctx, "git", "push", "-u", params.RemoteName, "HEAD"); err != nil {
		pushErr := fmt.Errorf("push to %q: %w", params.RemoteName, err)
		if rollbackErr := c.runIn(params.Dir, context.WithoutCancel(ctx), "git", "remote", "remove", params.RemoteName); rollbackErr != nil {
			return fmt.Errorf("%w; remove added remote: %v", pushErr, rollbackErr)
		}
		return pushErr
	}
	return nil
}

func PushNewRepo(ctx context.Context, params PushNewRepoParams) error {
	return defaultClient.PushNewRepo(ctx, params)
}
