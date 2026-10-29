package tangled

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// SearchHit is one record returned by org.tangled.search.query.
type SearchHit struct {
	URI   string         `json:"uri"`
	CID   string         `json:"cid"`
	NSID  string         `json:"nsid"`
	Score float64        `json:"score"`
	Value map[string]any `json:"value"`
}

// SearchResult contains the records matching a query.
type SearchResult struct {
	Hits   []SearchHit `json:"hits"`
	Cursor string      `json:"cursor,omitempty"`
}

// Search queries repository manifests, applying the type filter before the result limit.
func (t *Tangled) Search(ctx context.Context, query string, limit int64) (*SearchResult, error) {
	if limit < 0 {
		return nil, fmt.Errorf("search limit cannot be negative")
	}
	pageLimit := int64(100)
	if limit > 0 && limit < pageLimit {
		pageLimit = limit
	}
	hits, err := fetchAllPages(ctx, limit, func(ctx context.Context, cursor string) ([]SearchHit, *string, error) {
		params := map[string]any{"q": query, "nsid": RepoCollection, "limit": pageLimit}
		if cursor != "" {
			params["cursor"] = cursor
		}
		var page SearchResult
		if err := t.Client.Get(ctx, syntax.NSID("org.tangled.search.query"), params, &page); err != nil {
			return nil, nil, err
		}
		return page.Hits, &page.Cursor, nil
	})
	if err != nil {
		return nil, fmt.Errorf("search for %q: %w", query, err)
	}
	return &SearchResult{Hits: hits}, nil
}
