package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/bluesky-social/indigo/atproto/atclient"
)

func authenticatedATProto(ctx context.Context) (*atproto.ATProto, string, error) {
	client, did, err := auth.APIClient(ctx)
	if err != nil {
		if errors.Is(err, atproto.ErrNotAuthenticated) {
			return nil, "", fmt.Errorf("not logged in; run \"tg auth login\" first")
		}
		return nil, "", fmt.Errorf("resume auth session: %w", err)
	}
	return &atproto.ATProto{Client: client}, did.String(), nil
}

// publicAccountReader resolves handle to an unauthenticated client on its
// PDS for read-only queries of account-owned records (strings, public
// keys), returning the client and the owner's DID.
func publicAccountReader(ctx context.Context, handle string) (*atproto.ATProto, string, error) {
	ident, err := resolver.ResolveHandle(ctx, handle)
	if err != nil {
		return nil, "", fmt.Errorf("resolve handle %q: %w", handle, err)
	}

	pdsURL, err := resolver.ResolvePDS(ctx, ident.DID.String())
	if err != nil {
		return nil, "", fmt.Errorf("resolve PDS for %q: %w", handle, err)
	}

	return &atproto.ATProto{Client: &atclient.APIClient{Host: pdsURL}}, ident.DID.String(), nil
}
