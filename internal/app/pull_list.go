package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/tangled"
)

// ListPulls lists every pull request in the target repository.
func (s *Service) ListPulls(ctx context.Context, t Target) ([]Item, error) {
	repoDid, err := s.repoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	pulls, err := s.appview.ListPulls(ctx, repoDid, tangled.ListOpts{
		Limit: defaultListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list PRs for %q: %w", t.Repo, err)
	}
	return s.buildItems(ctx, pulls.Items, decodePull), nil
}
