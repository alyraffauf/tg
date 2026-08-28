package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/tangled"
)

// CommentIssue adds a comment to the issue identified by rkey.
func (s *Service) CommentIssue(ctx context.Context, t Target, rkey, body string) (*CreatedRecordResult, error) {
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
	issue, err := findByRKey(issues.Items, rkey, "issue")
	if err != nil {
		return nil, err
	}
	return s.createFeedComment(ctx, *issue, body, nil)
}
