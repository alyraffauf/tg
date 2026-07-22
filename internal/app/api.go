package app

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

// APIRequestInput describes an authenticated XRPC request.
type APIRequestInput struct {
	Endpoint syntax.NSID
	Method   string
	Fields   map[string]any
}

// APIResponse contains the response returned by an authenticated XRPC call.
type APIResponse struct {
	StatusCode int
	Body       []byte
}

// CallAPI performs an authenticated XRPC request for frontend-specific API
// commands.
func (s *Service) CallAPI(ctx context.Context, in APIRequestInput) (*APIResponse, error) {
	client, err := s.AuthenticatedAPIClient(ctx)
	if err != nil {
		return nil, err
	}
	request := atclient.NewAPIRequest(in.Method, in.Endpoint, nil)
	request.Headers.Set("Accept", "application/json")
	if in.Method == http.MethodGet {
		request.QueryParams = apiQuery(in.Fields)
	} else {
		body, err := json.Marshal(in.Fields)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
		request.Body = bytes.NewReader(body)
		request.Headers.Set("Content-Type", "application/json")
	}
	response, err := client.Do(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("call %s: %w", in.Endpoint, err)
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
