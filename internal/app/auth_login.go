package app

import (
	"context"
	"net/url"
)

// LoginWithPassword authenticates an account with an app password. When
// useInsecureFileStore is true, the session is stored in plaintext instead of the keyring.
func (s *Service) LoginWithPassword(ctx context.Context, identifier, password string, useInsecureFileStore bool) error {
	return s.auth.LoginWithPassword(ctx, identifier, password, useInsecureFileStore)
}

// StartLogin starts an OAuth login flow.
func (s *Service) StartLogin(ctx context.Context, identifier string) (string, error) {
	return s.auth.StartLogin(ctx, identifier)
}

// FinishLogin completes an OAuth login flow from callback query parameters.
func (s *Service) FinishLogin(ctx context.Context, query url.Values) error {
	return s.auth.FinishLogin(ctx, query)
}

// CancelLogin discards a pending OAuth login flow.
func (s *Service) CancelLogin() {
	s.auth.CancelLogin()
}
