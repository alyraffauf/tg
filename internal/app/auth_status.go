package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
)

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
