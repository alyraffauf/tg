package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type CreateStringInput struct {
	Filename    string
	Description string
	Contents    string
}
type stringRecord struct {
	Type        string `json:"$type"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"createdAt"`
	Snippet     struct {
		Type    string        `json:"$type"`
		Name    string        `json:"name"`
		Content *atproto.Blob `json:"content"`
	} `json:"snippet"`
}

func (s *Service) CreateString(ctx context.Context, in CreateStringInput) (*CreatedRecordResult, error) {
	if in.Filename == "" {
		return nil, fmt.Errorf("filename is required")
	}
	if len(in.Contents) > 1_000_000 {
		return nil, fmt.Errorf("string contents exceed 1,000,000 bytes")
	}
	pds, actor, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	blob, err := pds.UploadBlob(ctx, []byte(in.Contents), "text/plain")
	if err != nil {
		return nil, err
	}
	record := stringRecord{Type: tangled.StringCollection, Description: in.Description, CreatedAt: nowString()}
	record.Snippet.Type = tangled.StringCollection + "#textFile"
	record.Snippet.Name = in.Filename
	record.Snippet.Content = blob
	rkey := string(syntax.NewTIDNow(0))
	uri, cid, err := pds.CreateRecord(ctx, atproto.PutRecordInput{Repo: actor, Collection: tangled.StringCollection, Rkey: rkey, Record: record})
	if err != nil {
		return nil, err
	}
	var warnings []string
	if err := s.appview.AwaitRecord(ctx, uri, cid); err != nil {
		warnings = append(warnings, fmt.Sprintf("saved string, but indexing is pending: %v", err))
	}
	return &CreatedRecordResult{URI: uri, CID: cid, Rkey: rkey, Warnings: warnings}, nil
}
