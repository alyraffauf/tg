package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
)

// DeleteSSHKey removes a public key record from the authenticated user's account.
func (s *Service) DeleteSSHKey(ctx context.Context, rkey string) (*DeletedRecordResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	if err := atClient.DeleteRecord(ctx, atproto.DeleteRecordInput{
		Repo:       did,
		Collection: sshKeyCollection,
		Rkey:       rkey,
	}); err != nil {
		return nil, fmt.Errorf("delete SSH key: %w", err)
	}
	return &DeletedRecordResult{Rkey: rkey}, nil
}
