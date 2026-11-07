package app

import "context"

// ListPulls lists pull requests in the target repository.
func (s *Service) ListPulls(ctx context.Context, t Target, options ListOptions) (*ItemListResult, error) {
	listOptions, err := s.pullListOptions(ctx, options)
	if err != nil {
		return nil, err
	}
	return s.listTickets(ctx, t, options, listOptions, true)
}
