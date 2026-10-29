package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
)

// ListStrings lists every string owned by handle.
func (s *Service) ListStrings(ctx context.Context, handle string) (*StringListResult, error) {
	atClient, did, err := s.publicPDS(ctx, handle)
	if err != nil {
		return nil, err
	}
	records, err := atClient.ListAllRecords(ctx, did, stringCollection, atproto.ListRecordsOpts{Limit: defaultListLimit})
	if err != nil {
		return nil, fmt.Errorf("list strings for %q: %w", handle, err)
	}
	items, warnings := buildStringItems(records)
	return &StringListResult{Items: items, Warnings: warnings}, nil
}

func buildStringItems(records []atproto.RecordItem) ([]StringItem, []RecordWarning) {
	items := make([]StringItem, 0, len(records))
	var warnings []RecordWarning
	for _, rec := range records {
		str, err := decodeStringRecord(rec.Value)
		if err != nil {
			warnings = append(warnings, RecordWarning{URI: rec.URI, Error: err.Error()})
			continue
		}
		items = append(items, StringItem{
			Rkey:        extractRKey(rec.URI),
			URI:         rec.URI,
			Filename:    str.Snippet.Name,
			Description: str.Description,
			CreatedAt:   str.CreatedAt,
		})
	}
	return items, warnings
}
