package atproto

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/auth"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func hasOAuthScopes(granted, required []string) bool {
	scopes := make(map[string]bool, len(granted))
	for _, scope := range granted {
		scopes[normalizeOAuthScope(scope)] = true
	}
	for _, scope := range required {
		if !scopes[normalizeOAuthScope(scope)] {
			return false
		}
	}
	return true
}

func normalizeOAuthScope(scope string) string {
	// Servers may encode query parameters or use their named equivalents.
	if permission, err := auth.ParsePermissionString(scope); err == nil {
		return permission.ScopeString()
	}
	return scope
}

func (m *AuthManager) validateOAuthScopes(ctx context.Context, account Account, granted []string) error {
	if hasOAuthScopes(granted, DefaultScopes) {
		return nil
	}
	identifier := account.Handle
	if identifier == "" {
		identifier = account.DID
	}
	if err := m.store.DeleteSession(ctx, syntax.DID(account.DID), ""); err != nil {
		return fmt.Errorf("%w for %s; could not clear saved credentials: %w", ErrOAuthScopeMismatch, identifier, err)
	}
	return fmt.Errorf("%w; signed out %s; run \"tg auth login %s\" again", ErrOAuthScopeMismatch, identifier, identifier)
}
