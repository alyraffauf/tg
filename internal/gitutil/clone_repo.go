package gitutil

import (
	"context"
)

// CloneRepoParams groups the inputs to CloneRepo.
type CloneRepoParams struct {
	KnotHost string // Knot hosting the repository
	SSHPort  int    // Knot SSH port
	Handle   string // Tangled owner handle
	Repo     string // repository name
	RepoDir  string // local directory to clone into
}

// CloneRepo clones handle/repo from Tangled into params.RepoDir.
func (c *Client) CloneRepo(ctx context.Context, params CloneRepoParams) error {
	url := knotRemoteURL(params.KnotHost, params.SSHPort, params.Handle, params.Repo)
	return c.run(ctx, "git", "clone", url, params.RepoDir)
}

func CloneRepo(ctx context.Context, params CloneRepoParams) error {
	return defaultClient.CloneRepo(ctx, params)
}
