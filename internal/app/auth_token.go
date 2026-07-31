package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/bluesky-social/indigo/atproto/atclient"
)

// AccessToken returns the current session's access token, whether OAuth or
// app-password.
func (s *Service) AccessToken(ctx context.Context) (string, error) {
	session, err := s.auth.CurrentSession(ctx)
	if err == nil {
		token, _ := session.GetHostAccessData()
		if token == "" {
			return "", fmt.Errorf("current session has no access token")
		}
		return token, nil
	}
	if !errors.Is(err, atproto.ErrNotAuthenticated) {
		return "", fmt.Errorf("resume OAuth session: %w", err)
	}
	client, _, err := s.auth.APIClient(ctx)
	if err != nil {
		if errors.Is(err, atproto.ErrNotAuthenticated) {
			return "", fmt.Errorf("not logged in; run \"tg auth login\" first")
		}
		return "", fmt.Errorf("resume auth session: %w", err)
	}
	passwordAuth, ok := client.Auth.(*atclient.PasswordAuth)
	if !ok {
		return "", fmt.Errorf("not logged in; run \"tg auth login\" first")
	}
	token, _ := passwordAuth.GetTokens()
	if token == "" {
		return "", fmt.Errorf("current session has no access token")
	}
	return token, nil
}
