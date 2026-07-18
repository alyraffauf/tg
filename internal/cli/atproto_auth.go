package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
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
