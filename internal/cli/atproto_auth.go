package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
)

func requireAuthSession(ctx context.Context) (*oauth.ClientSession, error) {
	session, err := auth.CurrentSession(ctx)
	if err != nil {
		if errors.Is(err, atproto.ErrNotAuthenticated) {
			return nil, fmt.Errorf("not logged in; run \"tg auth login\" first")
		}
		return nil, fmt.Errorf("resume OAuth session: %w", err)
	}
	return session, nil
}

func authenticatedATProto(ctx context.Context) (*atproto.ATProto, string, error) {
	session, err := requireAuthSession(ctx)
	if err != nil {
		return nil, "", err
	}
	return &atproto.ATProto{Client: session.APIClient()}, session.Data.AccountDID.String(), nil
}
