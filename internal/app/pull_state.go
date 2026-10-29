package app

import (
	"context"
	"fmt"
)

func (s *Service) SetPullState(ctx context.Context, t Target, rkey, state string, expectedCID ...string) (*StateResult, error) {
	if state != "open" && state != "closed" {
		return nil, fmt.Errorf("unsupported state %q", state)
	}
	result, err := s.editTicket(ctx, ticketEditInput{ExpectedCID: firstCID(expectedCID), Target: t, Rkey: rkey, State: state, Pull: true})
	if err != nil {
		return nil, err
	}
	return &StateResult{Rkey: rkey, State: state, URI: result.URI, CID: result.CID, Warnings: result.Warnings}, nil
}
