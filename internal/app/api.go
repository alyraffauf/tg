package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// APIRequestInput describes an authenticated XRPC request.
type APIRequestInput struct {
	Endpoint string
	Method   string
	Fields   map[string]any
}

// APIResponse contains the response returned by an authenticated XRPC call.
type APIResponse = atproto.APIResponse

// CallAPI performs an authenticated XRPC request for frontend-specific API
// commands.
func (s *Service) CallAPI(ctx context.Context, in APIRequestInput) (*APIResponse, error) {
	endpoint, err := syntax.ParseNSID(in.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse NSID: %w", err)
	}
	client, err := s.sessions.APIClient(ctx)
	if err != nil {
		return nil, err
	}
	return atproto.CallAPI(ctx, client, in.Method, endpoint, in.Fields)
}
