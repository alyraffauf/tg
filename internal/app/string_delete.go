package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
)

// DeleteString removes a string record from the authenticated user's account.
func (s *Service) DeleteString(ctx context.Context, rkey string) (*DeletedRecordResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	if err := atClient.DeleteRecord(ctx, atproto.DeleteRecordInput{
		Repo:       did,
		Collection: stringCollection,
		Rkey:       rkey,
	}); err != nil {
		return nil, fmt.Errorf("delete string: %w", err)
	}
	return &DeletedRecordResult{Rkey: rkey}, nil
}
