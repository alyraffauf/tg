package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/tangled"
)

// ListIssues lists every issue in the target repository.
func (s *Service) ListIssues(ctx context.Context, t Target) ([]Item, error) {
	repoDid, err := s.repoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	issues, err := s.appview.ListIssues(ctx, repoDid, tangled.ListOpts{
		Limit: defaultListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list issues for %q: %w", t.Repo, err)
	}
	return s.buildItems(ctx, issues.Items, decodeIssue), nil
}
