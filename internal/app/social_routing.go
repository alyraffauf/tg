package app

import (
	"context"
	"fmt"
	"slices"

	"github.com/alyraffauf/tg/tangled"
)

func (s *Service) nativeSocial(ctx context.Context, repo *tangled.Repo) (bool, error) {
	description, err := s.knot.NewPublic(repo.Value.Knot).DescribeKnot(ctx)
	if err != nil {
		return false, fmt.Errorf("probe origin knot capabilities: %w", err)
	}
	return !description.Legacy && slices.Contains(description.Capabilities, "knot-social-writes"), nil
}
