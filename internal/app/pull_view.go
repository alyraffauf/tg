package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/tangled"
)

// ViewPull finds a single pull request by rkey within the target repository.
func (s *Service) ViewPull(ctx context.Context, t Target, rkey string) (*ViewResult, error) {
	repoDid, err := s.repoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	pulls, err := s.appview.ListPulls(ctx, repoDid, tangled.ListOpts{
		Limit: defaultListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list PRs for %s: %w", t, err)
	}
	found, err := findByRKey(pulls.Items, rkey, "pull request")
	if err != nil {
		return nil, err
	}
	decoded, err := decodePull(found.Value)
	if err != nil {
		return nil, fmt.Errorf("decode pull request %q: %w", rkey, err)
	}
	return &ViewResult{
		Rkey:         rkey,
		Title:        decoded.Title,
		State:        found.State,
		Body:         decoded.Body,
		Author:       s.resolveAuthor(ctx, extractDID(found.URI)),
		CreatedAt:    decoded.CreatedAt,
		SourceBranch: decoded.SourceBranch,
		TargetBranch: decoded.TargetBranch,
	}, nil
}
