package app

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
)

const maxPullPatchSize = 100 << 20

type PullPatch struct {
	URI          string
	Title        string
	Body         string
	TargetBranch string
	Patch        []byte
	Base         string
	Head         string
	CID          string
}

func (s *Service) PullPatch(ctx context.Context, t Target, rkey string) (*PullPatch, error) {
	repo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return nil, err
	}
	return s.pullPatch(ctx, repo, rkey)
}
func (s *Service) pullPatch(ctx context.Context, repo *tangled.Repo, rkey string) (*PullPatch, error) {
	did := stringValue(repo.Value.RepoDid)
	pulls, err := s.appview.ListPulls(ctx, did, tangled.ListOpts{Limit: 100})
	if err != nil {
		return nil, err
	}
	item, err := findByRKey(pulls.Items, rkey, "pull request")
	if err != nil {
		return nil, err
	}
	ticket, err := decodePull(item.Value)
	if err != nil {
		return nil, err
	}
	patches, err := s.appview.ListPatches(ctx, item.URI)
	if err != nil {
		return nil, err
	}
	if len(patches) == 0 {
		return nil, fmt.Errorf("pull request has no patch rounds")
	}
	last := patches[len(patches)-1]
	source := did
	if last.Value.Source != nil && last.Value.Source.Repo != "" {
		source = last.Value.Source.Repo
	}
	data, err := s.appview.FormatPatch(ctx, source, last.Value.Base.OID, last.Value.Head.OID)
	if err != nil {
		return nil, explainMirrorError(err)
	}
	return &PullPatch{URI: item.URI, CID: item.CID, Title: ticket.Title, Body: ticket.Body, TargetBranch: ticket.TargetBranch, Patch: data, Base: last.Value.Base.OID, Head: last.Value.Head.OID}, nil
}

func explainMirrorError(err error) error {
	var apiError *atclient.APIError
	if errors.As(err, &apiError) && apiError.Name == "RevisionNotFound" {
		return fmt.Errorf("the Git mirror has not caught up with this PR's commits; wait a moment and retry: %w", err)
	}
	return err
}
func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return data, err
}
