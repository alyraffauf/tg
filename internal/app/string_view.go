package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/internal/tangledlex"
)

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

func decodeStringRecord(value any) (tangledlex.String, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return tangledlex.String{}, fmt.Errorf("encode record: %w", err)
	}
	var record tangledlex.String
	if err := json.Unmarshal(data, &record); err != nil {
		return tangledlex.String{}, fmt.Errorf("decode record: %w", err)
	}
	return record, nil
}
