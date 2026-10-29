package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/alyraffauf/tg/tangled"
	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func (s *Service) createFeedComment(ctx context.Context, repo *tangled.Repo, item tangled.ListItem, body string) (*CreatedRecordResult, error) {
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("comment body must not be empty")
	}
	subject, err := commentSubjectFromListItem(item)
	if err != nil {
		return nil, err
	}
	if extractDID(item.URI) != stringValue(repo.Value.RepoDid) {
		return nil, fmt.Errorf("comment subject belongs to a different repository")
	}
	record := map[string]any{"$type": tangled.FeedCommentCollection, "subject": subject, "body": tangled.Markdown{Type: "org.tangled.markup.markdown", Text: body}, "createdAt": nowString()}
	receipt, warnings, err := s.socialProcedure(ctx, repo, "com.atproto.repo.createRecord", map[string]any{"collection": tangled.FeedCommentCollection, "record": record})
	if err != nil {
		return nil, err
	}
	return &CreatedRecordResult{URI: receipt.URI, CID: receipt.CID, Rkey: extractRKey(receipt.URI), Warnings: warnings}, nil
}
func commentSubjectFromListItem(item tangled.ListItem) (*comatproto.RepoStrongRef, error) {
	if _, err := syntax.ParseATURI(item.URI); err != nil {
		return nil, err
	}
	if _, err := syntax.ParseCID(item.CID); err != nil {
		return nil, fmt.Errorf("comment subject has no valid CID: %w", err)
	}
	return &comatproto.RepoStrongRef{LexiconTypeID: "com.atproto.repo.strongRef", Uri: item.URI, Cid: item.CID}, nil
}
