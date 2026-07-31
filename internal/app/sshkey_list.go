package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/tangledlex"
)

// ListSSHKeys lists every public key owned by handle.
func (s *Service) ListSSHKeys(ctx context.Context, handle string) ([]SSHKeyItem, error) {
	atClient, did, err := s.publicPDS(ctx, handle)
	if err != nil {
		return nil, err
	}
	records, err := atClient.ListAllRecords(ctx, did, sshKeyCollection, atproto.ListRecordsOpts{Limit: defaultListLimit})
	if err != nil {
		return nil, fmt.Errorf("list SSH keys for %q: %w", handle, err)
	}
	return buildSSHKeyItems(records), nil
}

func buildSSHKeyItems(records []atproto.RecordItem) []SSHKeyItem {
	items := make([]SSHKeyItem, 0, len(records))
	for _, rec := range records {
		var key tangledlex.PublicKey
		data, err := json.Marshal(rec.Value)
		if err != nil {
			continue
		}
		if err := json.Unmarshal(data, &key); err != nil {
			continue
		}
		items = append(items, SSHKeyItem{
			Name:      key.Name,
			Key:       key.Key,
			CreatedAt: key.CreatedAt,
			URI:       rec.URI,
		})
	}
	return items
}
