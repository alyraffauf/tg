package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/alyraffauf/tg/atproto"
	"github.com/bluesky-social/indigo/atproto/atclient"
)

var ErrNotAuthenticated = errors.New("not authenticated")

const (
	SessionStatusActive  = atproto.SessionStatusActive
	SessionStatusExpired = atproto.SessionStatusExpired
	SessionStatusUnknown = atproto.SessionStatusUnknown
)

// LoginWithPassword authenticates an account with an app password.
func (s *Service) LoginWithPassword(ctx context.Context, identifier, password string) error {
	return s.auth.LoginWithPassword(ctx, identifier, password)
}

// CurrentDID returns the DID for the active account.
func (s *Service) CurrentDID(ctx context.Context) (string, error) {
	did, err := s.auth.CurrentDID(ctx)
	if err != nil {
		return "", err
	}
	return did.String(), nil
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

func (s *Service) authenticatedPDS(ctx context.Context) (pdsClient, string, error) {
	return s.sessions.AuthenticatedPDS(ctx)
}

func (s *Service) publicPDS(ctx context.Context, handle string) (pdsClient, string, error) {
	return s.sessions.PublicPDS(ctx, handle)
}

// HandleOrSelf returns handle when non-empty, otherwise the authenticated
// user's handle.
func (s *Service) HandleOrSelf(ctx context.Context, handle string) (string, error) {
	if handle != "" {
		return handle, nil
	}
	did, err := s.auth.CurrentDID(ctx)
	if err != nil {
		if errors.Is(err, atproto.ErrNotAuthenticated) {
			return "", fmt.Errorf("not logged in; provide a handle or run \"tg auth login\"")
		}
		return "", fmt.Errorf("resume OAuth session: %w", err)
	}
	ident, err := s.resolver.ResolveDID(ctx, did.String())
	if err != nil {
		return "", fmt.Errorf("resolve your DID: %w", err)
	}
	return ident.Handle.String(), nil
}

// AuthStatus probes the active session. A missing session is reported as a
// zero AuthStatusResult (Authenticated=false), not an error.
func (s *Service) AuthStatus(ctx context.Context) (*AuthStatusResult, error) {
	status, did, err := s.auth.SessionStatus(ctx)
	if err != nil {
		if errors.Is(err, atproto.ErrNotAuthenticated) {
			return &AuthStatusResult{}, nil
		}
		return nil, fmt.Errorf("check session: %w", err)
	}
	author := s.resolveAuthor(ctx, did.String())
	return &AuthStatusResult{
		Authenticated: true,
		Status:        status,
		DID:           author.DID,
		Handle:        author.Handle,
	}, nil
}

// AuthAccounts lists all stored accounts, marking the active one.
func (s *Service) AuthAccounts(ctx context.Context) ([]AuthAccountResult, error) {
	accounts, activeDID, err := s.auth.Accounts()
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	results := make([]AuthAccountResult, 0, len(accounts))
	for _, account := range accounts {
		handle := account.Handle
		resolved := s.resolveAuthor(ctx, account.DID)
		if resolved.Handle != account.DID {
			handle = resolved.Handle
		}
		results = append(results, AuthAccountResult{
			Active: account.DID == activeDID,
			DID:    account.DID, Handle: handle, Method: account.Method,
		})
	}
	return results, nil
}

// SwitchAccount selects the active account by handle or DID.
func (s *Service) SwitchAccount(ctx context.Context, selector string) (*AuthAccountResult, error) {
	account, err := s.auth.SelectAccount(selector)
	if err != nil {
		return nil, fmt.Errorf("select account %q: %w", selector, err)
	}
	resolved := s.resolveAuthor(ctx, account.DID)
	return &AuthAccountResult{
		Active: true, DID: account.DID, Handle: resolved.Handle, Method: account.Method,
	}, nil
}

// Logout removes the active account's credentials (or all accounts when all
// is true). A missing session is reported as WasLoggedIn=false, not an error.
func (s *Service) Logout(ctx context.Context, all bool) (*AuthLogoutResult, error) {
	var err error
	if all {
		err = s.auth.LogoutAll(ctx)
	} else {
		err = s.auth.Logout(ctx)
	}
	if err != nil {
		if errors.Is(err, atproto.ErrNotAuthenticated) {
			return &AuthLogoutResult{WasLoggedIn: false}, nil
		}
		return nil, err
	}
	return &AuthLogoutResult{WasLoggedIn: true}, nil
}

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
