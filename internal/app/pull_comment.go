package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/tangled"
)

func (s *Service) CommentPull(ctx context.Context, t Target, rkey, body string) (*CreatedRecordResult, error) {
	did, err := s.repoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	pulls, err := s.appview.ListPulls(ctx, did, tangled.ListOpts{Limit: defaultListLimit})
	if err != nil {
		return nil, err
	}
	item, err := findByRKey(pulls.Items, rkey, "pull request")
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
	return s.createFeedComment(ctx, tangled.ListItem{URI: latest.URI, CID: latest.CID}, body, nil)
}
