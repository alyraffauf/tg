package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/knot"
)

func (s *Service) DeleteRepo(ctx context.Context, t Target) (*RepoDeleteResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.requireOwnedRepo(ctx, t, did)
	if err != nil {
		return nil, err
	}
	if repo.Value.Knot == "" {
		return nil, fmt.Errorf("repo %q has no knot", t.String())
	}
	rkey := extractRKey(repo.URI)
	existingRecord, getErr := atClient.GetRecord(ctx, did, repoCollection, rkey)
	var recordToRestore tangledlex.Repo
	if getErr == nil {
		data, err := json.Marshal(existingRecord.Value)
		if err != nil {
			return nil, fmt.Errorf("encode repository record for restore: %w", err)
		}
		if err := json.Unmarshal(data, &recordToRestore); err != nil {
			return nil, fmt.Errorf("decode repository record for restore: %w", err)
		}
		if err := tangledlex.ValidateRecord(repoCollection, recordToRestore); err != nil {
			return nil, fmt.Errorf("validate repository record for restore: %w", err)
		}
	}
	// getErr is non-fatal: the record may already be deleted. Only call
	// DeleteRecord if it still exists.

	token, err := atClient.GetServiceAuth(ctx, "did:web:"+repo.Value.Knot, "sh.tangled.repo.delete")
	if err != nil {
		return nil, fmt.Errorf("get knot authorization: %w", err)
	}
	if getErr == nil {
		if err := atClient.DeleteRecord(ctx, atproto.DeleteRecordInput{
			Repo:       did,
			Collection: repoCollection,
			Rkey:       rkey,
		}); err != nil {
			return nil, fmt.Errorf("delete repository record: %w", err)
		}
	}
	if err := s.knot.New(repo.Value.Knot, token).DeleteRepo(ctx, knot.DeleteRepoInput{
		DID:  did,
		Name: t.Repo,
		Rkey: rkey,
	}); err != nil {
		if getErr == nil {
			if _, _, restoreErr := atClient.PutRecord(ctx, atproto.PutRecordInput{
				Repo: did, Collection: repoCollection, Rkey: rkey, Record: recordToRestore,
			}); restoreErr != nil {
				return nil, fmt.Errorf("delete knot repository: %w; restore repository record: %v", err, restoreErr)
			}
		}
		return nil, err
	}
	return &RepoDeleteResult{URI: repo.URI}, nil
}
