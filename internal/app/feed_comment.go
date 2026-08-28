package app

import (
	"context"
	"fmt"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/tangled"
	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func (s *Service) createFeedComment(ctx context.Context, subjectItem tangled.ListItem, body string, pullRoundIdx *int64) (*CreatedRecordResult, error) {
	subject, err := commentSubjectFromListItem(subjectItem)
	if err != nil {
		return nil, err
	}

	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	rkey := string(syntax.NewTIDNow(0))
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: tangled.FeedCommentCollection,
		Rkey:       rkey,
		Record: tangledlex.FeedComment{
			LexiconTypeID: tangled.FeedCommentCollection,
			Subject:       subject,
			Body:          &tangledlex.FeedComment_Body{MarkupMarkdown: &tangledlex.MarkupMarkdown{Text: body, Original: &body}},
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			PullRoundIdx:  pullRoundIdx,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create feed comment: %w", err)
	}
	return &CreatedRecordResult{Rkey: rkey, URI: uri}, nil
}

func commentSubjectFromListItem(item tangled.ListItem) (*comatproto.RepoStrongRef, error) {
	if _, err := syntax.ParseATURI(item.URI); err != nil {
		return nil, fmt.Errorf("invalid comment subject URI %q: %w", item.URI, err)
	}
	if _, err := syntax.ParseCID(item.CID); err != nil {
		return nil, fmt.Errorf("invalid comment subject CID for %q: %w", item.URI, err)
	}
	return &comatproto.RepoStrongRef{Uri: item.URI, Cid: item.CID}, nil
}
