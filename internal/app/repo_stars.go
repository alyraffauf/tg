package app

import "context"

func (s *Service) repoStars(ctx context.Context, repoDID string) (int64, error) {
	return s.appview.CountStars(ctx, repoDID)
}
