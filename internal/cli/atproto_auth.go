package cli

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
)

func authenticatedATProto(ctx context.Context) (*atproto.ATProto, string, error) {
	if auth == nil || !auth.IsAuthenticated() {
		return nil, "", fmt.Errorf("not logged in; run \"tg auth login\" first")
	}

	pds, err := auth.APIClient(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("get auth client: %w", err)
	}
	return &atproto.ATProto{Client: pds}, auth.CurrentDID().String(), nil
}
