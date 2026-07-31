package tangled

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// CountStars returns the number of stars for a repository DID.
func (t *Tangled) CountStars(ctx context.Context, subject string) (int64, error) {
	var result struct {
		Count int64 `json:"count"`
	}
	if err := t.Client.Get(ctx, syntax.NSID("sh.tangled.feed.countStars"), map[string]any{"subject": subject}, &result); err != nil {
		return 0, fmt.Errorf("count stars for %q: %w", subject, err)
	}
	return result.Count, nil
}
