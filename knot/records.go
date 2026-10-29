package knot

import (
	"context"
	"errors"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type Description struct {
	DID          string   `json:"did"`
	Capabilities []string `json:"capabilities"`
	Legacy       bool     `json:"-"`
}
type Receipt struct {
	URI    string `json:"uri"`
	CID    string `json:"cid"`
	Change string `json:"x-tngl-change,omitempty"`
}

func (c *Client) DescribeKnot(ctx context.Context) (*Description, error) {
	var out Description
	err := c.Get(ctx, syntax.NSID("org.tangled.knot.describeKnot"), nil, &out)
	if err != nil {
		var e *atclient.APIError
		if !errors.As(err, &e) || e.StatusCode != 404 || (e.Name != "XRPCNotSupported" && e.Name != "MethodNotFound" && e.Name != "NotFound") {
			return nil, err
		}
		if err = c.Get(ctx, syntax.NSID("sh.tangled.knot.version"), nil, &out); err != nil {
			return nil, err
		}
		out.Legacy = true
	}
	if out.Capabilities == nil {
		return nil, fmt.Errorf("knot capability response is missing capabilities")
	}
	return &out, nil
}
func (c *Client) Procedure(ctx context.Context, method string, input, out any) error {
	return c.Post(ctx, syntax.NSID(method), input, out)
}
func (c *Client) GetRecord(ctx context.Context, repo, collection, rkey string) (*atproto.GetRecordOutput, error) {
	var out atproto.GetRecordOutput
	err := c.Get(ctx, syntax.NSID("com.atproto.repo.getRecord"), map[string]any{"repo": repo, "collection": collection, "rkey": rkey}, &out)
	return &out, err
}

// UploadBlob uploads an attachment using the client's repository-scoped service token.
func (c *Client) UploadBlob(ctx context.Context, data []byte, mimeType string) (*atproto.Blob, error) {
	return (&atproto.ATProto{Client: c.APIClient}).UploadBlob(ctx, data, mimeType)
}
