package app

import "context"

// EditPull patches a pull request's title and/or body. A nil pointer leaves
// the field untouched.
func (s *Service) EditPull(ctx context.Context, rkey string, title, body *string) error {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return err
	}
	return editRecord(ctx, atClient, did, pullCollection, rkey, title, body)
}
