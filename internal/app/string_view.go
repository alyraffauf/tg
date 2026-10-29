package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func (s *Service) ViewString(ctx context.Context, handle, rkey string) (*StringViewResult, error) {
	pds, actor, err := s.publicPDS(ctx, handle)
	if err != nil {
		return nil, err
	}
	found, err := pds.GetRecord(ctx, actor, tangled.StringCollection, rkey)
	if err != nil {
		return nil, err
	}
	record, err := decodeStringRecord(found.Value)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(record.Snippet.Content.Ref)
	if err != nil {
		return nil, err
	}
	var ref struct {
		Link string `json:"$link"`
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		return nil, err
	}
	if _, err := syntax.ParseCID(ref.Link); err != nil {
		return nil, fmt.Errorf("invalid string blob: %w", err)
	}
	host, err := s.resolver.ResolvePDS(ctx, actor)
	if err != nil {
		return nil, err
	}
	contents, err := atproto.NewPublic(host, s.httpClient).GetBlob(ctx, actor, ref.Link)
	if err != nil {
		return nil, err
	}
	if len(contents) > 1_000_000 {
		return nil, fmt.Errorf("string blob exceeds 1,000,000 bytes")
	}
	return &StringViewResult{Rkey: rkey, URI: found.URI, Filename: record.Snippet.Name, Author: Author{DID: actor, Handle: handle}, Description: record.Description, Contents: string(contents), CreatedAt: record.CreatedAt}, nil
}
func decodeStringRecord(value any) (stringRecord, error) {
	var record stringRecord
	data, err := json.Marshal(value)
	if err != nil {
		return record, err
	}
	if err = json.Unmarshal(data, &record); err != nil {
		return record, err
	}
	if record.Type != tangled.StringCollection || record.Snippet.Type != tangled.StringCollection+"#textFile" || record.Snippet.Name == "" || record.Snippet.Content == nil {
		return record, fmt.Errorf("invalid text string record")
	}
	return record, nil
}
