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

// CommentIssue adds a comment to the issue identified by rkey.
func (s *Service) CommentIssue(ctx context.Context, t Target, rkey, body string) (*CreatedRecordResult, error) {
	repoDid, err := s.repoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	issues, err := s.appview.ListIssues(ctx, repoDid, tangled.ListOpts{
		Limit: defaultListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list issues for %s: %w", t, err)
	}
	issue, err := findByRKey(issues.Items, rkey, "issue")
	if err != nil {
		return nil, err
	}
	return s.createIssueComment(ctx, issue.URI, body)
}

func (s *Service) createIssueComment(ctx context.Context, issueURI, body string) (*CreatedRecordResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	rkey := string(syntax.NewTIDNow(0))
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: issueCollection + ".comment",
		Rkey:       rkey,
		Record: tangledlex.RepoIssueComment{
			LexiconTypeID: issueCollection + ".comment",
			Issue:         issueURI,
			Body:          body,
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create issue comment: %w", err)
	}
	return &CreatedRecordResult{Rkey: rkey, URI: uri}, nil
}
