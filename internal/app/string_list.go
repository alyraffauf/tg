package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/tangledlex"
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
		var str tangledlex.String
		data, err := json.Marshal(rec.Value)
		if err != nil {
			warnings = append(warnings, RecordWarning{URI: rec.URI, Error: fmt.Sprintf("encode string record: %v", err)})
			continue
		}
		if err := json.Unmarshal(data, &str); err != nil {
			warnings = append(warnings, RecordWarning{URI: rec.URI, Error: fmt.Sprintf("decode string record: %v", err)})
			continue
		}
		// Records without a filename are not strings; skip them rather
		// than rendering a blank row.
		if str.Filename == "" {
			warnings = append(warnings, RecordWarning{URI: rec.URI, Error: "string record has no filename"})
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
	return items, warnings
}
