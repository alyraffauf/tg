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
	RepoDID  string // stable repository DID, when available
	RepoDir  string // local directory to clone into
}

// CloneRepo clones the repository identified by params into params.RepoDir.
func (c *Client) CloneRepo(ctx context.Context, params CloneRepoParams) error {
	url, err := cloneRemoteURL(params)
	if err != nil {
		return err
	}
	return c.run(ctx, "git", "clone", "--", url, params.RepoDir)
}

func CloneRepo(ctx context.Context, params CloneRepoParams) error {
	return defaultClient.CloneRepo(ctx, params)
}

func cloneRemoteURL(params CloneRepoParams) (string, error) {
	if params.RepoDID != "" {
		return repositoryDIDRemoteURL(repositoryDIDRemote{
			Protocol: params.Protocol,
			KnotHost: params.KnotHost,
			SSHPort:  params.SSHPort,
			RepoDID:  params.RepoDID,
		})
	}
	switch params.Protocol {
	case "ssh":
		return knotRemoteURL(params.KnotHost, params.SSHPort, params.Handle+"/"+params.Repo), nil
	case "https":
		if params.KnotHost == "" {
			return "", fmt.Errorf("HTTPS clone requires a Knot host")
		}
		return "https://" + params.KnotHost + "/" + params.Handle + "/" + params.Repo + ".git", nil
	default:
		return "", fmt.Errorf("unsupported clone protocol %q", params.Protocol)
	}
}
