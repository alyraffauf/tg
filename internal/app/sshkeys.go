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

// AddSSHKey writes a new sh.tangled.publicKey record.
func (s *Service) AddSSHKey(ctx context.Context, name, key string) (*SSHKeyAddResult, error) {
	atClient, did, err := s.AuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: sshKeyCollection,
		Rkey:       string(syntax.NewTIDNow(0)),
		Record: tangled.SSHKeyRecord{
			Type:      sshKeyCollection,
			Key:       key,
			Name:      name,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("add SSH key: %w", err)
	}
	return &SSHKeyAddResult{Name: name, URI: uri}, nil
}

// ListSSHKeys lists every public key owned by handle.
func (s *Service) ListSSHKeys(ctx context.Context, handle string) ([]SSHKeyItem, error) {
	atClient, did, err := s.PublicAccountReader(ctx, handle)
	if err != nil {
		return nil, err
	}
	records, err := atClient.ListAllRecords(ctx, did, sshKeyCollection, atproto.ListRecordsOpts{Limit: defaultListLimit})
	if err != nil {
		return nil, fmt.Errorf("list SSH keys for %q: %w", handle, err)
	}
	return buildSSHKeyItems(records), nil
}

// DeleteSSHKey removes a public key record from the authenticated user's account.
func (s *Service) DeleteSSHKey(ctx context.Context, rkey string) (*DeletedRecordResult, error) {
	atClient, did, err := s.AuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	if err := atClient.DeleteRecord(ctx, atproto.DeleteRecordInput{
		Repo:       did,
		Collection: sshKeyCollection,
		Rkey:       rkey,
	}); err != nil {
		return nil, fmt.Errorf("delete SSH key: %w", err)
	}
	return &DeletedRecordResult{Rkey: rkey}, nil
}

func buildSSHKeyItems(records []atproto.RecordItem) []SSHKeyItem {
	items := make([]SSHKeyItem, 0, len(records))
	for _, rec := range records {
		var key tangled.SSHKeyRecord
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
