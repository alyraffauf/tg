package atproto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// APIResponse contains the response returned by an authenticated XRPC call.
type APIResponse struct {
	StatusCode int
	Body       []byte
}

// CallAPI performs an authenticated XRPC request.
func CallAPI(ctx context.Context, client *atclient.APIClient, method string, endpoint syntax.NSID, fields map[string]any) (*APIResponse, error) {
	request := atclient.NewAPIRequest(method, endpoint, nil)
	request.Headers.Set("Accept", "application/json")
	if method == http.MethodGet {
		request.QueryParams = apiQuery(fields)
	} else {
		body, err := json.Marshal(fields)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
		request.Body = bytes.NewReader(body)
		request.Headers.Set("Content-Type", "application/json")
	}
	response, err := client.Do(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("call %s: %w", endpoint, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read API response: %w", err)
	}
	return &APIResponse{StatusCode: response.StatusCode, Body: body}, nil
}

func apiQuery(fields map[string]any) map[string][]string {
	query := make(map[string][]string, len(fields))
	for key, value := range fields {
		query[key] = []string{fmt.Sprint(value)}
	}
	return query
}
