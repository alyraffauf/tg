package app

import (
	"context"
	"errors"

	"github.com/alyraffauf/tg/atproto"
)

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
