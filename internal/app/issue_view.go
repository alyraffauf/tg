package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/tangled"
)

// ViewIssue finds a single issue by rkey within the target repository.
func (s *Service) ViewIssue(ctx context.Context, t Target, rkey string) (*ViewResult, error) {
	repoDid, err := s.repoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	issues, err := s.appview.ListIssues(ctx, repoDid, tangled.ListOpts{
		Limit: defaultListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list issues for %s: %w", t, err)
	}
	found, err := findByRKey(issues.Items, rkey, "issue")
	if err != nil {
		return nil, err
	}
	decoded, err := decodeIssue(found.Value)
	if err != nil {
		return nil, fmt.Errorf("decode issue %q: %w", rkey, err)
	}
	return &ViewResult{
		Rkey:      rkey,
		Title:     decoded.Title,
		State:     found.State,
		Body:      decoded.Body,
		Author:    s.resolveAuthor(ctx, extractDID(found.URI)),
		CreatedAt: decoded.CreatedAt,
	}, nil
}
