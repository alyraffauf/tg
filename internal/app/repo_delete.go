package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
)

func (s *Service) DeleteRepo(ctx context.Context, t Target) (*RepoDeleteResult, error) {
	pds, actor, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.requireOwnedRepo(ctx, t, actor)
	if err != nil {
		return nil, err
	}
	key := strings.ToLower(stringValue(repo.Value.Name))
	found, err := pds.GetRecord(ctx, actor, tangled.DeclarationCollection, key)
	if err != nil {
		return nil, err
	}
	if found.CID == nil {
		return nil, fmt.Errorf("declaration response has no CID")
	}
	data, err := json.Marshal(found.Value)
	if err != nil {
		return nil, err
	}
	var declaration struct {
		Repo string `json:"repo"`
	}
	if err := json.Unmarshal(data, &declaration); err != nil {
		return nil, err
	}
	if declaration.Repo != stringValue(repo.Value.RepoDid) {
		return nil, fmt.Errorf("repository declaration changed while preparing deletion; inspect it before retrying")
	}
	token, err := pds.GetServiceAuth(ctx, serviceDID(repo.Value.Knot), "org.tangled.repo.deleteRepo")
	if err != nil {
		return nil, err
	}
	if err := pds.DeleteRecord(ctx, atproto.DeleteRecordInput{Repo: actor, Collection: tangled.DeclarationCollection, Rkey: key, SwapRecord: found.CID}); err != nil {
		return nil, err
	}
	if err := s.knot.New(repo.Value.Knot, token).DeleteRepo(ctx, knot.DeleteRepoInput{Repo: stringValue(repo.Value.RepoDid)}); err != nil {
		if isDefiniteRejection(err) {
			if _, _, restore := pds.CreateRecord(ctx, atproto.PutRecordInput{Repo: actor, Collection: tangled.DeclarationCollection, Rkey: key, Record: found.Value}); restore != nil {
				return nil, fmt.Errorf("delete repository: %w; restore declaration: %v", err, restore)
			}
		}
		return nil, fmt.Errorf("repository deletion was not confirmed: %w; inspect the repository before retrying", err)
	}
	return &RepoDeleteResult{URI: repo.URI}, nil
}
func isRecordNotFound(err error) bool {
	var e *atclient.APIError
	if !errors.As(err, &e) {
		return false
	}
	return (e.Name == "RecordNotFound" && (e.StatusCode == 400 || e.StatusCode == 404)) || (e.Name == "NotFound" && e.StatusCode == 404)
}
