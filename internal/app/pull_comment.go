package app

import (
	"context"
	"fmt"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// CommentPull adds a comment to the pull request identified by rkey.
func (s *Service) CommentPull(ctx context.Context, t Target, rkey, body string) (*CreatedRecordResult, error) {
	repoDid, err := s.repoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	pulls, err := s.appview.ListPulls(ctx, repoDid, tangled.ListOpts{
		Limit: defaultListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list PRs for %s: %w", t, err)
	}
	pull, err := findByRKey(pulls.Items, rkey, "pull request")
	if err != nil {
		return nil, err
	}
	return s.createPullComment(ctx, pull.URI, body)
}

func (s *Service) createPullComment(ctx context.Context, pullURI, body string) (*CreatedRecordResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	rkey := string(syntax.NewTIDNow(0))
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: pullCollection + ".comment",
		Rkey:       rkey,
		Record: tangledlex.RepoPullComment{
			LexiconTypeID: pullCollection + ".comment",
			Pull:          pullURI,
			Body:          body,
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create pull request comment: %w", err)
	}
	return &CreatedRecordResult{Rkey: rkey, URI: uri}, nil
}
