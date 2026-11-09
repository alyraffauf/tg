package app

import "context"

func (s *Service) CommentPull(ctx context.Context, t Target, rkey, body string) (*CreatedRecordResult, error) {
	repo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return nil, err
	}
	item, _, err := s.readTicket(ctx, repo, rkey, true)
	if err != nil {
		return nil, err
	}
	return s.createFeedComment(ctx, repo, *item, body)
}
