package app

import "context"

// EditIssue patches an issue's title and/or body. A nil pointer leaves the
// field untouched.
func (s *Service) EditIssue(ctx context.Context, rkey string, title, body *string) error {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return err
	}
	return editRecord(ctx, atClient, did, issueCollection, rkey, title, body)
}
