package app

import (
	"context"
	"fmt"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// CreateIssue writes a new issue record in the target repository.
func (s *Service) CreateIssue(ctx context.Context, t Target, title, body string) (*CreatedRecordResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	repoDid, err := s.repoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	rkey := string(syntax.NewTIDNow(0))
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: issueCollection,
		Rkey:       rkey,
		Record: tangledlex.RepoIssue{
			LexiconTypeID: issueCollection,
			Repo:          repoDid,
			Title:         title,
			Body:          optionalString(body),
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create issue: %w", err)
	}
	return &CreatedRecordResult{Rkey: rkey, URI: uri}, nil
}
