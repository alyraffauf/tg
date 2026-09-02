package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/tangledlex"
)

// ListSSHKeys lists every public key owned by handle.
func (s *Service) ListSSHKeys(ctx context.Context, handle string) (*SSHKeyListResult, error) {
	atClient, did, err := s.publicPDS(ctx, handle)
	if err != nil {
		return nil, err
	}
	records, err := atClient.ListAllRecords(ctx, did, sshKeyCollection, atproto.ListRecordsOpts{Limit: defaultListLimit})
	if err != nil {
		return nil, fmt.Errorf("list SSH keys for %q: %w", handle, err)
	}
	items, warnings := buildSSHKeyItems(records)
	return &SSHKeyListResult{Items: items, Warnings: warnings}, nil
}

func buildSSHKeyItems(records []atproto.RecordItem) ([]SSHKeyItem, []RecordWarning) {
	items := make([]SSHKeyItem, 0, len(records))
	var warnings []RecordWarning
	for _, rec := range records {
		var key tangledlex.PublicKey
		data, err := json.Marshal(rec.Value)
		if err != nil {
			warnings = append(warnings, RecordWarning{URI: rec.URI, Error: fmt.Sprintf("encode SSH key record: %v", err)})
			continue
		}
		if err := json.Unmarshal(data, &key); err != nil {
			warnings = append(warnings, RecordWarning{URI: rec.URI, Error: fmt.Sprintf("decode SSH key record: %v", err)})
			continue
		}
		items = append(items, SSHKeyItem{
			Name:      key.Name,
			Key:       key.Key,
			CreatedAt: key.CreatedAt,
			URI:       rec.URI,
		})
	}
	return items, warnings
}
