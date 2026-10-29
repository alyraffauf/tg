package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/tangled"
)

func (s *Service) CommentPull(ctx context.Context, t Target, rkey, body string) (*CreatedRecordResult, error) {
	repo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return nil, err
	}
	item, _, err := s.readTicket(ctx, repo, rkey, true)
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
	latest := patches[len(patches)-1]
	return s.createFeedComment(ctx, repo, tangled.ListItem{URI: latest.URI, CID: latest.CID}, body)
}
