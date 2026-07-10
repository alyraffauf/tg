package atproto

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// ATProto wraps an atproto PDS client. The client may be authenticated (from
// AuthManager.APIClient) or a plain public client for read-only queries.
type ATProto struct {
	Client *atclient.APIClient
}

type PutRecordInput struct {
	Repo       string `json:"repo"`
	Collection string `json:"collection"`
	Rkey       string `json:"rkey"`
	Record     any    `json:"record"`
}

type GetRecordOutput struct {
	URI   string `json:"uri"`
	CID   string `json:"cid,omitempty"`
	Value any    `json:"value"`
}

// RecordItem is a single record in a listRecords response.
type RecordItem struct {
	URI   string `json:"uri"`
	CID   string `json:"cid,omitempty"`
	Value any    `json:"value"`
}

type ListRecordsOutput struct {
	Records []RecordItem `json:"records"`
	Cursor  *string      `json:"cursor,omitempty"`
}

type ListRecordsOpts struct {
	Limit   int64
	Cursor  string
	Reverse bool
}

// maxRecordPages caps how many pages ListAllRecords will follow, as a
// safety net against a server that never returns an empty cursor.
const maxRecordPages = 1000

// PutRecord writes a record to the PDS, returning its at:// URI and CID.
func (a *ATProto) PutRecord(ctx context.Context, in PutRecordInput) (uri, cid string, err error) {
	var out struct {
		URI string `json:"uri"`
		CID string `json:"cid,omitempty"`
	}
	if err := a.Client.Post(ctx, syntax.NSID("com.atproto.repo.putRecord"), in, &out); err != nil {
		return "", "", fmt.Errorf("put %s/%s record: %w", in.Collection, in.Rkey, err)
	}
	return out.URI, out.CID, nil
}

func (a *ATProto) GetRecord(ctx context.Context, repo, collection, rkey string) (*GetRecordOutput, error) {
	var out GetRecordOutput
	params := map[string]any{
		"repo":       repo,
		"collection": collection,
		"rkey":       rkey,
	}
	if err := a.Client.Get(ctx, syntax.NSID("com.atproto.repo.getRecord"), params, &out); err != nil {
		return nil, fmt.Errorf("get %s/%s record for %q: %w", collection, rkey, repo, err)
	}
	return &out, nil
}

func (a *ATProto) ListRecords(ctx context.Context, repo, collection string, opts ListRecordsOpts) (*ListRecordsOutput, error) {
	params := map[string]any{
		"repo":       repo,
		"collection": collection,
	}
	if opts.Limit > 0 {
		params["limit"] = opts.Limit
	}
	if opts.Cursor != "" {
		params["cursor"] = opts.Cursor
	}
	if opts.Reverse {
		params["reverse"] = true
	}
	var out ListRecordsOutput
	if err := a.Client.Get(ctx, syntax.NSID("com.atproto.repo.listRecords"), params, &out); err != nil {
		return nil, fmt.Errorf("list %s records for %q: %w", collection, repo, err)
	}
	return &out, nil
}

// ListAllRecords fetches every record in collection for repo, following
// pagination cursors until the listing is exhausted. opts.Limit sets the
// page size; opts.Cursor is ignored since pagination always starts from
// the first page.
func (a *ATProto) ListAllRecords(ctx context.Context, repo, collection string, opts ListRecordsOpts) ([]RecordItem, error) {
	var all []RecordItem
	cursor := ""

	for range maxRecordPages {
		opts.Cursor = cursor
		out, err := a.ListRecords(ctx, repo, collection, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, out.Records...)

		if out.Cursor == nil || *out.Cursor == "" {
			return all, nil
		}
		cursor = *out.Cursor
	}

	return nil, fmt.Errorf("exceeded %d pages listing %s records for %q", maxRecordPages, collection, repo)
}
