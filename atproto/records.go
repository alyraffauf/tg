package atproto

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// PutRecordInput is the argument to a com.atproto.repo.putRecord call.
type PutRecordInput struct {
	Repo       string `json:"repo"`
	Collection string `json:"collection"`
	Rkey       string `json:"rkey"`
	Record     any    `json:"record"`
}

// PutRecord writes a record to the PDS, returning its at:// URI and CID. The
// record must include its $type field.
func PutRecord(ctx context.Context, pds *atclient.APIClient, in PutRecordInput) (uri, cid string, err error) {
	if pds == nil {
		return "", "", fmt.Errorf("PDS client is required")
	}
	var out struct {
		URI string `json:"uri"`
		CID string `json:"cid,omitempty"`
	}
	if err := pds.Post(ctx, syntax.NSID("com.atproto.repo.putRecord"), in, &out); err != nil {
		return "", "", fmt.Errorf("put %s/%s record: %w", in.Collection, in.Rkey, err)
	}
	return out.URI, out.CID, nil
}
