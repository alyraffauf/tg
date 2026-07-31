package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
)

var ErrNotAuthenticated = errors.New("not authenticated")

const (
	SessionStatusActive  = atproto.SessionStatusActive
	SessionStatusExpired = atproto.SessionStatusExpired
	SessionStatusUnknown = atproto.SessionStatusUnknown
)

func (s *Service) authenticatedPDS(ctx context.Context) (pdsClient, string, error) {
	return s.sessions.AuthenticatedPDS(ctx)
}

func (s *Service) publicPDS(ctx context.Context, handle string) (pdsClient, string, error) {
	return s.sessions.PublicPDS(ctx, handle)
}

// CurrentDID returns the DID for the active account.
func (s *Service) CurrentDID(ctx context.Context) (string, error) {
	did, err := s.auth.CurrentDID(ctx)
	if err != nil {
		return "", err
	}
	return did.String(), nil
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
