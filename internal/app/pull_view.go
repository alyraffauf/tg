package app

import "context"

func (s *Service) ViewPull(ctx context.Context, t Target, rkey string) (*ViewResult, error) {
	return s.viewTicket(ctx, t, rkey, true)
}
