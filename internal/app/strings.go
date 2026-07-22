package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
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
		Record: tangled.StringRecord{
			Type:        stringCollection,
			Filename:    in.Filename,
			Description: in.Description,
			Contents:    in.Contents,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create string: %w", err)
	}
	return &CreatedRecordResult{Rkey: rkey, URI: uri}, nil
}

// ListStrings lists every string owned by handle.
func (s *Service) ListStrings(ctx context.Context, handle string) ([]StringItem, error) {
	atClient, did, err := s.publicPDS(ctx, handle)
	if err != nil {
		return nil, err
	}
	records, err := atClient.ListAllRecords(ctx, did, stringCollection, atproto.ListRecordsOpts{Limit: defaultListLimit})
	if err != nil {
		return nil, fmt.Errorf("list strings for %q: %w", handle, err)
	}
	return buildStringItems(records), nil
}

// ViewString fetches a single string by rkey from handle's account.
func (s *Service) ViewString(ctx context.Context, handle, rkey string) (*StringViewResult, error) {
	atClient, did, err := s.publicPDS(ctx, handle)
	if err != nil {
		return nil, err
	}
	found, err := atClient.GetRecord(ctx, did, stringCollection, rkey)
	if err != nil {
		return nil, fmt.Errorf("get string %q for %q: %w", rkey, handle, err)
	}
	record, err := decodeStringRecord(found.Value)
	if err != nil {
		return nil, fmt.Errorf("decode string %q: %w", rkey, err)
	}
	return &StringViewResult{
		Rkey:        rkey,
		URI:         found.URI,
		Filename:    record.Filename,
		Author:      Author{DID: did, Handle: handle},
		Description: record.Description,
		Contents:    record.Contents,
		CreatedAt:   record.CreatedAt,
	}, nil
}

// DeleteString removes a string record from the authenticated user's account.
func (s *Service) DeleteString(ctx context.Context, rkey string) (*DeletedRecordResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	if err := atClient.DeleteRecord(ctx, atproto.DeleteRecordInput{
		Repo:       did,
		Collection: stringCollection,
		Rkey:       rkey,
	}); err != nil {
		return nil, fmt.Errorf("delete string: %w", err)
	}
	return &DeletedRecordResult{Rkey: rkey}, nil
}

func buildStringItems(records []atproto.RecordItem) []StringItem {
	items := make([]StringItem, 0, len(records))
	for _, rec := range records {
		var str tangled.StringRecord
		data, err := json.Marshal(rec.Value)
		if err != nil {
			continue
		}
		if err := json.Unmarshal(data, &str); err != nil {
			continue
		}
		// Records without a filename are not strings; skip them rather
		// than rendering a blank row.
		if str.Filename == "" {
			continue
		}
		items = append(items, StringItem{
			Rkey:        extractRKey(rec.URI),
			URI:         rec.URI,
			Filename:    str.Filename,
			Description: str.Description,
			CreatedAt:   str.CreatedAt,
		})
	}
	return items
}

func decodeStringRecord(value any) (tangled.StringRecord, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return tangled.StringRecord{}, fmt.Errorf("encode record: %w", err)
	}
	var record tangled.StringRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return tangled.StringRecord{}, fmt.Errorf("decode record: %w", err)
	}
	return record, nil
}
