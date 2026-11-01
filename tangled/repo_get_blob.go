package tangled

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

const maxBlobSize = 10 << 20

func (t *Tangled) GetBlob(ctx context.Context, repoDID, path string) ([]byte, error) {
	request := atclient.NewAPIRequest(http.MethodGet, syntax.NSID("org.tangled.temp.git.getBlob"), nil)
	request.QueryParams = map[string][]string{"repo": {repoDID}, "path": {path}}

	response, err := t.Client.Do(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("get blob %q: %w", path, err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var body atclient.ErrorBody
		if decodeErr := json.NewDecoder(response.Body).Decode(&body); decodeErr == nil && body.Name != "" {
			return nil, fmt.Errorf("get blob %q: %w", path, body.APIError(response.StatusCode))
		}
		return nil, fmt.Errorf("get blob %q: HTTP %d", path, response.StatusCode)
	}

	contents, err := io.ReadAll(io.LimitReader(response.Body, maxBlobSize+1))
	if err != nil {
		return nil, fmt.Errorf("read blob %q: %w", path, err)
	}
	if int64(len(contents)) > maxBlobSize {
		return nil, fmt.Errorf("read blob %q: response exceeds %d bytes", path, maxBlobSize)
	}
	return contents, nil
}

// IsNotFound reports whether err is an appview 404.
func IsNotFound(err error) bool {
	var apiErr *atclient.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}
