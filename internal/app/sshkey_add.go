package app

import (
	"context"
	"fmt"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// AddSSHKey writes a new sh.tangled.publicKey record.
func (s *Service) AddSSHKey(ctx context.Context, name, key string) (*SSHKeyAddResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: sshKeyCollection,
		Rkey:       string(syntax.NewTIDNow(0)),
		Record: tangledlex.PublicKey{
			LexiconTypeID: sshKeyCollection,
			Key:           key,
			Name:          name,
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("add SSH key: %w", err)
	}
	return &SSHKeyAddResult{Name: name, URI: uri}, nil
}
