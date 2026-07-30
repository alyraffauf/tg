package gitutil

import (
	"context"
	"fmt"
)

// CloneRepoParams groups the inputs to CloneRepo.
type CloneRepoParams struct {
	KnotHost string // Knot hosting the repository
	SSHPort  int    // Knot SSH port
	Protocol string // SSH or HTTPS
	Handle   string // Tangled owner handle
	Repo     string // repository name
	RepoDir  string // local directory to clone into
}

// CloneRepo clones handle/repo from Tangled into params.RepoDir.
func (c *Client) CloneRepo(ctx context.Context, params CloneRepoParams) error {
	url, err := cloneRemoteURL(params.Protocol, params.KnotHost, params.SSHPort, params.Handle, params.Repo)
	if err != nil {
		return err
	}
	return c.run(ctx, "git", "clone", url, params.RepoDir)
}

func CloneRepo(ctx context.Context, params CloneRepoParams) error {
	return defaultClient.CloneRepo(ctx, params)
}

func cloneRemoteURL(protocol, knotHost string, sshPort int, handle, repo string) (string, error) {
	switch protocol {
	case "ssh":
		return knotRemoteURL(knotHost, sshPort, handle, repo), nil
	case "https":
		if knotHost == "" {
			return "", fmt.Errorf("HTTPS clone requires a Knot host")
		}
		return "https://" + knotHost + "/" + handle + "/" + repo + ".git", nil
	default:
		return "", fmt.Errorf("unsupported clone protocol %q", protocol)
	}
}
