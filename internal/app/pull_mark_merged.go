package app

import (
	"context"
	"fmt"
)

type MarkPullMergedInput struct {
	Target      Target
	Rkey        string
	ExpectedCID string
}

// MarkPullMerged records completion for code integrated outside tg.
func (s *Service) MarkPullMerged(ctx context.Context, input MarkPullMergedInput) (*StateResult, error) {
	updated, err := s.editTicket(ctx, ticketEditInput{Target: input.Target, Rkey: input.Rkey, State: pullStateCompleted, Pull: true, ExpectedCID: input.ExpectedCID})
	if err != nil {
		return nil, fmt.Errorf("record manual merge: %w", err)
	}
	return &StateResult{Rkey: input.Rkey, State: "merged", URI: updated.URI, CID: updated.CID, Warnings: updated.Warnings}, nil
}
