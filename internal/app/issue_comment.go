package app

import "context"

// CommentIssue adds a comment to the issue identified by rkey.
func (s *Service) CommentIssue(ctx context.Context, t Target, rkey, body string) (*CreatedRecordResult, error) {
	repo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return nil, err
	}
	issue, _, err := s.readTicket(ctx, repo, rkey, false)
	if err != nil {
		return nil, err
	}
	return s.createFeedComment(ctx, repo, *issue, body)
}
