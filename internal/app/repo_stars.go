package app

import "context"

func (s *Service) repoStars(ctx context.Context, repoDID string) int64 {
	stars, _ := s.appview.CountStars(ctx, repoDID)
	return stars
}
