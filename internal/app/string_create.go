package app

import (
	"context"
	"fmt"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// CreateStringInput configures string creation.
type CreateStringInput struct {
	Filename    string
	Description string
	Contents    string
}

// CreateString writes a new sh.tangled.string record.
func (s *Service) CreateString(ctx context.Context, in CreateStringInput) (*CreatedRecordResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	rkey := string(syntax.NewTIDNow(0))
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: stringCollection,
		Rkey:       rkey,
		Record: tangledlex.String{
			LexiconTypeID: stringCollection,
			Filename:      in.Filename,
			Description:   in.Description,
			Contents:      in.Contents,
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create string: %w", err)
	}
	return &CreatedRecordResult{Rkey: rkey, URI: uri}, nil
}
