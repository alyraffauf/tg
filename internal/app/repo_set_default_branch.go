package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/knot"
)

// SetRepoDefaultBranch sets the default branch of the authenticated user's repo.
func (s *Service) SetRepoDefaultBranch(ctx context.Context, t Target, branch string) (*RepoDefaultBranchResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.requireOwnedRepo(ctx, t, did)
	if err != nil {
		return nil, err
	}
	if repo.Value.Knot == "" {
		return nil, fmt.Errorf("repo %q has no knot", t.String())
	}
	if err := s.setKnotDefaultBranch(ctx, atClient, repo.Value.Knot, repo.URI, branch); err != nil {
		return nil, err
	}
	return &RepoDefaultBranchResult{URI: repo.URI, Branch: branch}, nil
}

func (s *Service) setKnotDefaultBranch(ctx context.Context, atClient pdsClient, knotHost, repoURI, branch string) error {
	token, err := atClient.GetServiceAuth(ctx, "did:web:"+knotHost, "sh.tangled.repo.setDefaultBranch")
	if err != nil {
		return fmt.Errorf("get knot authorization: %w", err)
	}
	return s.knot.New(knotHost, token).SetDefaultBranch(ctx, knot.SetDefaultBranchInput{
		Repo:          repoURI,
		DefaultBranch: branch,
	})
}
