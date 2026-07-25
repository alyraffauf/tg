package tangled

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type lexClient struct {
	client *atclient.APIClient
}

func (c lexClient) LexDo(ctx context.Context, method, inputEncoding, endpoint string, params map[string]any, bodyData, out any) error {
	if method != "GET" || inputEncoding != "" || bodyData != nil {
		return fmt.Errorf("unsupported Bobbin lexicon request %s %s", method, endpoint)
	}

	nsid, err := syntax.ParseNSID(endpoint)
	if err != nil {
		return fmt.Errorf("parse Bobbin endpoint: %w", err)
	}
	return c.client.Get(ctx, nsid, params, out)
}

func (t *Tangled) lexClient() lexClient {
	return lexClient{client: t.Client}
}
