package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/tangledlex"
)

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

func buildStringItems(records []atproto.RecordItem) []StringItem {
	items := make([]StringItem, 0, len(records))
	for _, rec := range records {
		var str tangledlex.String
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
