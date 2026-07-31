package app

import "context"

// SetIssueState closes or reopens an issue. state is the bare verb
// ("open" or "closed").
func (s *Service) SetIssueState(ctx context.Context, t Target, rkey, state string) (*StateResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	target, _, err := s.targetRecord(ctx, t, issueCollection, rkey)
	if err != nil {
		return nil, err
	}
	if err := putState(ctx, atClient, did, rkey, issueCollection, target, state); err != nil {
		return nil, err
	}
	return &StateResult{Rkey: rkey, State: state}, nil
}
