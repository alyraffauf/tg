package tangled

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// SearchHit is one record returned by sh.tangled.search.query.
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

// Search queries Bobbin's indexed records.
func (t *Tangled) Search(ctx context.Context, query string, limit int64) (*SearchResult, error) {
	params := map[string]any{"q": query}
	if limit > 0 {
		params["limit"] = limit
	}
	var result SearchResult
	if err := t.Client.Get(ctx, syntax.NSID("sh.tangled.search.query"), params, &result); err != nil {
		return nil, fmt.Errorf("search for %q: %w", query, err)
	}
	return &result, nil
}
