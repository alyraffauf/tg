package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/tangled"
)

// CommentPull adds a comment to the pull request identified by rkey.
func (s *Service) CommentPull(ctx context.Context, t Target, rkey, body string) (*CreatedRecordResult, error) {
	repoDid, err := s.repoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	pulls, err := s.appview.ListPulls(ctx, repoDid, tangled.ListOpts{
		Limit: defaultListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list PRs for %s: %w", t, err)
	}
	pull, err := findByRKey(pulls.Items, rkey, "pull request")
	if err != nil {
		return nil, err
	}
	var record tangledlex.RepoPull
	if err := json.Unmarshal(pull.Value, &record); err != nil {
		return nil, fmt.Errorf("decode pull request %q: %w", rkey, err)
	}
	if len(record.Rounds) == 0 {
		return nil, fmt.Errorf("pull request %q has no rounds", rkey)
	}
	latestRoundIdx := int64(len(record.Rounds) - 1)
	return s.createFeedComment(ctx, *pull, body, &latestRoundIdx)
}
