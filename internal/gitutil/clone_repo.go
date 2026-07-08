package gitutil

import (
	"context"
)

// CloneRepoParams groups the inputs to CloneRepo.
type CloneRepoParams struct {
	Handle  string // Tangled owner handle
	Repo    string // repository name
	RepoDir string // local directory to clone into
}

// CloneRepo clones handle/repo from Tangled into params.RepoDir.
func CloneRepo(ctx context.Context, params CloneRepoParams) error {
	url := tangledRemoteURL(params.Handle, params.Repo)
	return run(ctx, "git", "clone", url, params.RepoDir)
}
