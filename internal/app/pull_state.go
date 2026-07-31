package app

import "context"

// SetPullState closes or reopens a pull request. status is the bare verb
// ("open" or "closed").
func (s *Service) SetPullState(ctx context.Context, t Target, rkey, status string) (*StateResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	target, _, err := s.targetRecord(ctx, t, pullCollection, rkey)
	if err != nil {
		return nil, err
	}
	if err := putState(ctx, atClient, did, rkey, pullCollection, target, status); err != nil {
		return nil, err
	}
	return &StateResult{Rkey: rkey, State: status}, nil
}
