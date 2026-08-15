package app

import (
	"context"
	"fmt"
)

// ListPulls lists pull requests in the target repository.
func (s *Service) ListPulls(ctx context.Context, t Target, options ListOptions) ([]Item, error) {
	repoDid, err := s.repoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	listOptions, err := s.pullListOptions(ctx, options)
	if err != nil {
		return nil, err
	}
	pulls, err := s.appview.ListPulls(ctx, repoDid, listOptions)
	if err != nil {
		return nil, fmt.Errorf("list PRs for %q: %w", t.Repo, err)
	}
	return s.buildItems(ctx, pulls.Items, decodePull), nil
}
