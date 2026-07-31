package atproto

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

const maxBlobSize = 100 << 20

// NewPublic returns an unauthenticated PDS client.
func NewPublic(host string, httpClient *http.Client) *ATProto {
	return &ATProto{Client: &atclient.APIClient{Client: httpClient, Host: host}}
}

// GetBlob downloads a blob from a PDS.
func (a *ATProto) GetBlob(ctx context.Context, did, cid string) ([]byte, error) {
	request := atclient.NewAPIRequest(http.MethodGet, syntax.NSID("com.atproto.sync.getBlob"), nil)
	request.QueryParams = map[string][]string{"did": {did}, "cid": {cid}}
	response, err := a.Client.Do(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("get blob: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("get blob: PDS returned HTTP %d", response.StatusCode)
	}
	contents, err := io.ReadAll(io.LimitReader(response.Body, maxBlobSize+1))
	if err != nil {
		return nil, fmt.Errorf("read blob: %w", err)
	}
	if int64(len(contents)) > maxBlobSize {
		return nil, fmt.Errorf("read blob: response exceeds %d bytes", maxBlobSize)
	}
	return contents, nil
}
